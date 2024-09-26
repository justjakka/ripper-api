package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"ripper-api/ripper"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

func returnError(err error, c echo.Context) error {
	msg := &Message{
		Msg: err.Error(),
	}
	return c.JSON(http.StatusInternalServerError, msg)
}

func checkUrl(url string) (string, string) {
	pat := regexp.MustCompile(`^(?:https:\/\/(?:beta\.music|music)\.apple\.com\/(\w{2})(?:\/album|\/album\/.+))\/(?:id)?(\d[^\D]+)(?:$|\?)`)
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
	aval_queues, err := insp.Queues()

	if err != nil {
		return returnError(err, c)
	}

	min_queue, min := 0, 0

	for _, val := range aval_queues {
		info, err := insp.GetQueueInfo(val)
		if err != nil {
			return returnError(err, c)
		}

		if info.Active < min {
			min_queue, err = strconv.Atoi(val)
			min = info.Active
			if err != nil {
				return returnError(err, c)
			}
		}
	}

	task, err := ripper.NewRipTask(storefront, albumId, cc.ServerConfig.WebDir, cc.Wrappers[min_queue])
	if err != nil {
		return returnError(err, c)
	}

	info, err := cc.Client.Enqueue(task, asynq.Retention(time.Hour))
	if err != nil {
		return returnError(err, c)
	}
	msg := &Message{
		Msg: info.ID,
	}
	return c.JSON(http.StatusAccepted, msg)
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
	info, err := insp.GetTaskInfo("default", job.JobId)
	if err != nil {
		return returnError(err, c)
	}

	if info.State == 1 {
		msg := &Message{
			Msg: info.State.String(),
		}
		return c.JSON(http.StatusProcessing, msg)
	} else if info.State == 2 || info.State == 3 || info.State == 7 {
		msg := &Message{
			Msg: info.State.String(),
		}
		return c.JSON(http.StatusTooEarly, msg)
	} else if info.State == 6 {
		var p ripper.RipPayload

		if err := json.Unmarshal(info.Payload, &p); err != nil {
			return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
		}

		meta, err := ripper.GetMeta(p.AlbumId, p.Token, p.Storefront)

		if err != nil {
			return err
		}
		zipName := fmt.Sprintf("%s - %s.zip", meta.Data[0].Attributes.ArtistName, meta.Data[0].Attributes.Name)
		sanZipName := ripper.ForbiddenNames.ReplaceAllString(zipName, "_")

		msg := &Message{
			Msg: sanZipName,
		}

		return c.JSON(http.StatusOK, msg)
	} else {
		msg := &Message{
			Msg: info.LastErr,
		}
		return c.JSON(http.StatusInternalServerError, msg)
	}
}

func SendFile(c echo.Context) error {
	url := new(SubmittedUrl)
	cc := c.(*ConfigContext)
	if err := c.Bind(url); err != nil {
		msg := &Message{
			Msg: err.Error(),
		}
		return c.JSON(http.StatusBadRequest, msg)
	}
	if err := c.Validate(url); err != nil {
		return err
	}

	sanZipName := filepath.Join(cc.WebDir, url.Url)

	return c.File(sanZipName)
}
