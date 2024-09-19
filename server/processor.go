package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type Test struct {
	Url string `json:"url" xml:"url"`
	Str string `json:"str" xml:"str"`
}

func QueryRedis(RequestId string, c echo.Context) (Job, error) {

	ctx := context.Background()
	cc := c.(*ConfigContext)
	listenAddr := fmt.Sprintf(":%d", cc.ServerConfig.PortRedis)
	rdb := redis.NewClient(&redis.Options{
		Addr:     listenAddr,
		Password: cc.ServerConfig.RedisPw,
		DB:       0, // use default DB
	})

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

func Unmarshal(val string) {
	panic("unimplemented")
}

func ProcessLink(c echo.Context) error {
	cc := c.(*ConfigContext)
	port := cc.ServerConfig.PortRedis
	listenPort := fmt.Sprintf(":%d", port)
	requrl := c.Param("urlhex")
	test := &Test{Url: requrl, Str: listenPort}
	return c.JSON(http.StatusOK, test)
}

func ProcessRequestID(c echo.Context) error {
	requrl := c.Param("reqid")
	fmt.Println("reached", requrl)
	test := &Test{Url: requrl, Str: "test"}
	return c.JSON(http.StatusOK, test)
}
