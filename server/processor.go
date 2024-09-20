package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"ripper-api/ripper"

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

	return c.JSON(http.StatusAccepted, info.ID)
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

	if info.State == 1 {
		return c.JSON(http.StatusProcessing, info.State)
	} else if info.State == 2 || info.State == 3 {
		return c.JSON(http.StatusTooEarly, info.State)
	} else if info.State == 5 || info.State == 4 {
		return c.JSON(http.StatusInternalServerError, info.LastErr)
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
		sanZipName := filepath.Join(p.WebDir, ripper.ForbiddenNames.ReplaceAllString(zipName, "_"))

		return c.File(sanZipName)
	} else {
		return c.JSON(http.StatusOK, info.State)
	}
}
