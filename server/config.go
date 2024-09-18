package server

import (
	"github.com/labstack/echo/v4"
)

type ServerConfig struct {
	BindAddr    string
	WebDir      string
	BindWrapper string
	KeyList     *[]string
}

type ApplicationContext struct {
	echo.Context
	ServerConfig
}

type KeyDB struct {
	Name string `json:"name" xml:"name"`
	Key  string `json:"key" xml:"key"`
}
