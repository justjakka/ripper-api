package server

import "github.com/labstack/echo/v4"

type ServerConfig struct {
	Port        uint
	PortRedis   uint
	PortWrapper uint
	WebDir      string
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
