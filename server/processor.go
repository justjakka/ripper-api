package server

import (
	"fmt"
	"net/http"
	"regexp"
	"ripper-api/ripper"
	"time"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

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
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(url); err != nil {
		return err
	}
	storefront, albumId := checkUrl(url.Url)

	if storefront == "" && albumId == "" {
		msg := fmt.Sprintf("Invalid link: %v", url.Url)
		return c.JSON(http.StatusBadRequest, msg)
	}

	task, err := ripper.NewRipTask(storefront, albumId, cc.ServerConfig.PortWrapper, cc.ServerConfig.WebDir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	info, err := cc.Client.Enqueue(task, asynq.Retention(time.Hour))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, info.ID)
}

func ProcessRequestID(c echo.Context) error {
	cc := c.(*ConfigContext)
	job := new(JobQuery)
	if err := c.Bind(job); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(job); err != nil {
		return err
	}
	insp := cc.Inspector
	info, err := insp.GetTaskInfo("default", job.JobId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	if info.State != 6 {
		return c.JSON(http.StatusOK, info.State)
	} else {
		return c.JSON(http.StatusOK, "release downloaded")
	}
}
