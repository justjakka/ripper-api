package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type Test struct {
	Str1 string `json:"str1"`
	Str2 string `json:"str2"`
}

func QueryRedis(c echo.Context, RequestId string) (Job, error) {

	ctx := context.Background()
	cc := c.(*ConfigContext)
	rdb := cc.Client

	val, err := rdb.Get(ctx, RequestId).Bytes()

	if err == redis.Nil {
		return Job{}, nil
	} else if err != nil {
		return Job{}, err
	}
	var response Job

	err = json.Unmarshal(val, &response)

	if err != nil {
		return Job{}, err
	}

	return response, nil
}

func RedisSet(c echo.Context, RequestId string, job *Job) error {
	ctx := context.Background()
	cc := c.(*ConfigContext)

	rdb := cc.Client

	p, err := json.Marshal(job)

	if err != nil {
		return err
	}

	err = rdb.Set(ctx, RequestId, p, time.Hour).Err()

	if err != nil {
		return err
	}
	return nil
}

func ProcessLink(c echo.Context) error {

	newJob := Job{Status: 0, Message: "Job created"}
	err := RedisSet(c, c.Response().Header().Get(echo.HeaderXRequestID), &newJob)

	if err != nil {
		return err
	}

	test := &Test{Str1: c.Response().Header().Get(echo.HeaderXRequestID), Str2: "is ok"}
	return c.JSON(http.StatusOK, test)
}

func ProcessRequestID(c echo.Context) error {
	job, err := QueryRedis(c, c.Param("reqid"))

	if err != nil {
		return err
	}

	if job.Message == "" {
		return c.JSON(http.StatusBadRequest, "")
	} else {
		status := fmt.Sprintf("%d", job.Status)
		test := &Test{Str1: status, Str2: job.Message}
		return c.JSON(http.StatusOK, test)
	}

}
