package server

import "github.com/labstack/echo/v4"

type ServerConfig struct {
	BindAddr    string
	WebDir      string
	BindWrapper string
	BindRedis   string
	RedisPw     string
	KeyList     *[]string
}

type ConfigContext struct {
	echo.Context
	*ServerConfig
}

type Job struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
