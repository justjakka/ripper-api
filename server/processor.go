package server

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
	"net"

	"ripper-api/ripper"

	"github.com/hibiken/asynq"
)

func createZipBuffer(albumFolder string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	archiveWriter := zip.NewWriter(buf)

	defer func(archiveWriter *zip.Writer) {
		_ = archiveWriter.Close()
	}(archiveWriter)

	err := filepath.WalkDir(albumFolder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		filePath := filepath.Base(path)

		w, err := archiveWriter.Create(filePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(w, f)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return buf, nil
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
			return returnError(err, c)
		}

		if info.Active < minTasks {
			queuename = i
			minTasks = info.Active
		}
	}

	task, err := ripper.NewRipTask(storefront, albumId, cc.Config.WebDir, cc.Wrappers[queuename])
	if err != nil {
		return returnError(err, c)
	}

	info, err := cc.Client.Enqueue(task, asynq.Retention(time.Hour), asynq.Queue(fmt.Sprintf("%v", queuename)))
	if err != nil {
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
		return returnError(err, c)
	}
	
	switch info.State {
		case 1:
			return c.NoContent(http.StatusNoContent)

		case 2, 3, 7:
			return c.NoContent(http.StatusCreated)

		case 6:
			buf, err := createZipBuffer(string(info.Result))
			if err != nil {
				msg := &Message{
					Msg: err.Error(),
				}
				return c.JSON(http.StatusInternalServerError, msg)
			}
			zipReader := io.Reader(buf)

			task, err := ripper.NewDeleteTask(string(info.Result))
			if err != nil {
				return returnError(err, c)
			}

			_, err = cc.Client.Enqueue(task, asynq.Queue(info.Queue), asynq.ProcessIn(time.Hour))
			if err != nil {
				return returnError(err, c)
			}

			c.Response().Header().Set(echo.HeaderContentLength, strconv.Itoa(buf.Len()))
		
			return StreamConnWrapper(c, http.StatusOK, "application/zip", zipReader)

		default:
			msg := &Message{
				Msg: info.LastErr,
			}
			return c.JSON(http.StatusInternalServerError, msg)
	}
}

func StreamConnWrapper(c echo.Context, status int, contentType string, r io.Reader) error {
    err := c.Stream(status, contentType, r)
    if err != nil {
        opErr, ok := err.(*net.OpError)
        if ok && opErr.Op == "write" && opErr.Err.Error() == "connection reset by peer" {
            return nil
        }
        c.Logger().Errorf("streaming error: %w", err)
    }
    return nil
}

