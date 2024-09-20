package server

import (
	"net/http"
	"regexp"
	"ripper-api/ripper"

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
		return c.JSON(http.StatusBadRequest, "Invalid link")
	}

	task, err := ripper.NewRipTask(storefront, albumId, cc.ServerConfig.PortWrapper, cc.ServerConfig.WebDir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	info, err := cc.Client.Enqueue(task)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, info.State)
}
