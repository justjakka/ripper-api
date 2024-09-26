package server

import (
	"github.com/go-playground/validator"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

type ServerConfig struct {
	Port         uint
	AddressRedis string
	Wrappers     []string
	WebDir       string
	RedisPw      string
	KeyList      []string
}

type ConfigContext struct {
	echo.Context
	*ServerConfig
	*asynq.Client
	*asynq.Inspector
}

type (
	JobQuery struct {
		JobId string `json:"jobid" validate:"required"`
	}

	SubmittedUrl struct {
		Url string `json:"url" validate:"required"`
	}

	CustomValidator struct {
		validator *validator.Validate
	}
)

type Message struct {
	Msg string `json:"message"`
}
