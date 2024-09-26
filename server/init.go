package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"ripper-api/ripper"

	"github.com/go-playground/validator"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func createEcho(config *ServerConfig, logger zerolog.Logger, asynqClient *asynq.Client, asynqInspector *asynq.Inspector) *echo.Echo {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &ConfigContext{c, config, asynqClient, asynqInspector}
			return next(cc)
		}
	})

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				logger.Info().
					Str("URI", v.URI).
					Str("IP", v.RemoteIP).
					Int("status", v.Status).
					Str("method", v.Method).
					Msg("request")
			} else {
				logger.Error().
					Str("URI", v.URI).
					Str("IP", v.RemoteIP).
					Int("status", v.Status).
					Str("method", v.Method).
					Err(v.Error).
					Msg("request")
			}
			return nil
		},
	}))

	e.HideBanner = true

	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Pre(middleware.AddTrailingSlash())

	e.Validator = &CustomValidator{validator: validator.New()}

	e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:api-key",
		Validator: func(key string, c echo.Context) (bool, error) {
			for _, line := range config.KeyList {
				if key == line {
					return true, nil
				}
			}
			return false, nil
		},
	}))

	e.POST("/", ProcessLink)
	e.GET("/job/", ProcessRequestID)
	e.GET("/file/", SendFile)

	return e
}

func CreateEchoWithServer(ctx context.Context, config *ServerConfig) (*echo.Echo, *http.Server) {
	logger := zerolog.Ctx(ctx)

	asynqClient := asynq.NewClient(&asynq.RedisClientOpt{
		Addr:     config.AddressRedis,
		Password: config.RedisPw,
		DB:       0,
	})

	for i := range len(config.Wrappers) {
		task, err := ripper.NewInitQueueTask()
		if err != nil {
			panic(err)
		}
		_, err = asynqClient.Enqueue(task, asynq.Queue(fmt.Sprintf("%v", i)))
		if err != nil {
			panic(err)
		}
		msg := fmt.Sprintf("Queue %d initialized...", i)
		logger.Info().Msg(msg)
	}

	asynqInspector := asynq.NewInspector(&asynq.RedisClientOpt{
		Addr:     config.AddressRedis,
		Password: config.RedisPw,
		DB:       0,
	})

	e := createEcho(config, logger.With().Logger(), asynqClient, asynqInspector)

	listenAddr := fmt.Sprintf(":%d", config.Port)

	srv := &http.Server{
		Addr:        listenAddr,
		Handler:     e,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}

	return e, srv
}
