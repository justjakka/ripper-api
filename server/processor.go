package server

import (
	"fmt"
	"net/http"
	"regexp"
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
	min := 100
	var queuename int
	for i := range len(cc.Wrappers) {
		info, err := insp.GetQueueInfo(fmt.Sprintf("%v", i))
		if err != nil {
			return returnError(err, c)
		}

		if info.Active < min {
			queuename = i
			min = info.Active
		}
	}

	task, err := ripper.NewRipTask(storefront, albumId, cc.ServerConfig.WebDir, cc.Wrappers[queuename])
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

	if info.State == 1 {
		return c.NoContent(http.StatusNoContent)
	} else if info.State == 2 || info.State == 3 || info.State == 7 {
		return c.NoContent(http.StatusCreated)
	} else if info.State == 6 {
		return c.File(string(info.Result))
	} else {
		msg := &Message{
			Msg: info.LastErr,
		}
		return c.JSON(http.StatusInternalServerError, msg)
	}
}
