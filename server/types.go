package server

import (
	"github.com/go-playground/validator"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

type Config struct {
	Port         uint
	Address      string
	AddressRedis string
	Wrappers     []string
	WebDir       string
	RedisPw      string
	KeyList      []string
}

type ConfigContext struct {
	echo.Context
	*Config
	*asynq.Client
	*asynq.Inspector
}

type (
	JobQuery struct {
		JobId   string `json:"jobid" validate:"required"`
		QueueId string `json:"queueid" validate:"required"`
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
