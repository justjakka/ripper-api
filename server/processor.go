package server

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func QueryRedis(c echo.Context, RequestId string) (string, error) {

	ctx := context.Background()
	cc := c.(*ConfigContext)
	rdb := cc.Client

	response, err := rdb.Get(ctx, RequestId).Result()

	if err != nil {
		return "", err
	}

	return response, nil
}

func RedisSet(c echo.Context, RequestId string, message string) error {
	ctx := context.Background()
	cc := c.(*ConfigContext)
	rdb := cc.Client

	err := rdb.Set(ctx, RequestId, message, time.Hour).Err()

	if err != nil {
		return err
	}

	return nil
}

func ProcessLink(c echo.Context) error {

	err := RedisSet(c, c.Response().Header().Get(echo.HeaderXRequestID), "Job created")

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, c.Response().Header().Get(echo.HeaderXRequestID))
}

func ProcessRequestID(c echo.Context) error {
	response, err := QueryRedis(c, c.Param("reqid"))

	if err != nil {
		return err
	}

	if response == "" {
		return c.JSON(http.StatusBadRequest, "No response")
	} else {
		return c.JSON(http.StatusOK, response)
	}

}
