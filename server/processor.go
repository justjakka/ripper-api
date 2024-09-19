package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Test struct {
	Url string `json:"url" xml:"url"`
	Str string `json:"str" xml:"str"`
}

func ProcessLink(c echo.Context) error {
	requrl := c.Param("urlhex")
	fmt.Println("reached", requrl)
	test := &Test{Url: requrl, Str: c.Response().Header().Get(echo.HeaderXRequestID)}
	return c.JSON(http.StatusOK, test)
}

func ProcessRequestID(c echo.Context) error {
	requrl := c.Param("urlhex")
	fmt.Println("reached", requrl)
	test := &Test{Url: requrl, Str: "test"}
	return c.JSON(http.StatusOK, test)
}
