package server

import (
	"archive/zip"
	"compress/flate"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"ripper-api/ripper"

	"github.com/hibiken/asynq"
)

func writeZip(path fs.FS, oWriter io.Writer) error {
	zipWriter := zip.NewWriter(oWriter)
	defer zipWriter.Close()

	zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})

	err := fs.WalkDir(path, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("zip: cannot add non-regular file")
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = name
		h.Method = zip.Deflate

		fw, err := zipWriter.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := path.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

func returnError(err error, c echo.Context) error {
	msg := &Message{
		Msg: err.Error(),
	}
	return c.JSON(http.StatusInternalServerError, msg)
}

func checkUrl(url string) (string, string) {
	pat := regexp.MustCompile(`^https://(?:beta\.music|music)\.apple\.com/(\w{2})(?:/album|/album/.+)/(?:id)?(\d+)(?:$|\?)`)
	matches := pat.FindAllStringSubmatch(url, -1)
	if matches == nil {
		return "", ""
	} else {
		return matches[0][1], matches[0][2]
	}
}

func ProcessLink(c echo.Context) error {
	cc := c.(*ConfigContext)

	url := new(SubmittedUrl)

	if err := c.Bind(url); err != nil {
		msg := &Message{
			Msg: err.Error(),
		}
		return c.JSON(http.StatusBadRequest, msg)
	}

	if err := c.Validate(url); err != nil {
		return err
	}

	storefront, albumId := checkUrl(url.Url)

	if storefront == "" && albumId == "" {
		msg := &Message{
			Msg: fmt.Sprintf("Invalid link: %v", url.Url),
		}
		return c.JSON(http.StatusBadRequest, msg)
	}

	insp := cc.Inspector
	minTasks := 100
	var queuename int
	for i := range len(cc.Wrappers) {
		info, err := insp.GetQueueInfo(fmt.Sprintf("%v", i))
		if err != nil {
			c.Logger().Errorf("failed to get queue info: %v", err)
			return returnError(err, c)
		}

		if info.Active < minTasks {
			queuename = i
			minTasks = info.Active
		}
	}

	task, err := ripper.NewRipTask(storefront, albumId, cc.Config.WebDir, cc.Wrappers[queuename])
	if err != nil {
		c.Logger().Errorf("failed to create new rip task: %v", err)
		return returnError(err, c)
	}

	info, err := cc.Client.Enqueue(task, asynq.Retention(time.Hour), asynq.Queue(fmt.Sprintf("%v", queuename)))
	if err != nil {
		c.Logger().Errorf("failed to enqueue task: %v", err)
		return returnError(err, c)
	}
	return c.JSON(http.StatusAccepted, JobQuery{JobId: info.ID, QueueId: info.Queue})
}

func ProcessRequestID(c echo.Context) error {
	cc := c.(*ConfigContext)

	job := new(JobQuery)

	if err := c.Bind(job); err != nil {
		msg := &Message{
			Msg: err.Error(),
		}
		return c.JSON(http.StatusBadRequest, msg)
	}

	if err := c.Validate(job); err != nil {
		return err
	}

	insp := cc.Inspector

	info, err := insp.GetTaskInfo(job.QueueId, job.JobId)
	if err != nil {
		c.Logger().Errorf("failed to get task info: %v", err)
		return returnError(err, c)
	}

	switch info.State {
	case 1:
		return c.NoContent(http.StatusNoContent)

	case 2, 3, 7:
		return c.NoContent(http.StatusCreated)

	case 6:
		task, err := ripper.NewDeleteTask(string(info.Result))
		if err != nil {
			c.Logger().Errorf("failed to create new delete task: %v", err)
			return returnError(err, c)
		}

		_, err = cc.Client.Enqueue(task, asynq.Queue(info.Queue), asynq.ProcessIn(time.Hour))
		if err != nil {
			c.Logger().Errorf("failed to enqueue delete task: %v", err)
			return returnError(err, c)
		}

		pr, pw := io.Pipe()
		go func() {
			if err := writeZip(os.DirFS(string(info.Result)), pw); err != nil {
				c.Logger().Errorf("error on writing zip: %v", err)
			}

			if err := pw.Close(); err != nil {
				c.Logger().Errorf("error on closing zip writer: %v", err)
			}
		}()

		return StreamConnWrapper(c, http.StatusOK, "application/zip", pr)

	default:
		c.Logger().Errorf("error: %v", err)
		return returnError(err, c)
	}
}

func StreamConnWrapper(c echo.Context, status int, contentType string, r io.Reader) error {
	err := c.Stream(status, contentType, r)
	if err != nil {
		var opErr *net.OpError
		ok := errors.As(err, &opErr)
		if ok && opErr.Op == "write" && opErr.Err.Error() == "connection reset by peer" {
			return nil
		}
		c.Logger().Errorf("streaming error: %v", err)
	}
	return nil
}
