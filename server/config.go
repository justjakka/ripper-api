package server

import (
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

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
	*redis.Client
}

type Job struct {
	Status  uint8  `json:"status"` // 0 - done; 1 - in process; 2 - error
	Message string `json:"message"`
}
