package server

import (
	"github.com/go-playground/validator"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

type ServerConfig struct {
	Port         uint
	AddressRedis string
	PortRedis    uint
	PortWrapper  uint
	WebDir       string
	RedisPw      string
	KeyList      *[]string
}

type ConfigContext struct {
	echo.Context
	*ServerConfig
	*asynq.Client
}

type (
	SubmittedUrl struct {
		Url string `json:"url" validate:"required"`
	}

	CustomValidator struct {
		validator *validator.Validate
	}
)
