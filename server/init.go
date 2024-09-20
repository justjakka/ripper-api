package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
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
					Int("status", v.Status).
					Str("method", v.Method).
					Msg("request")
			} else {
				logger.Error().
					Str("URI", v.URI).
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
			for _, line := range *config.KeyList {
				if key == line {
					return true, nil
				}
			}
			return false, nil
		},
	}))

	e.POST("/", ProcessLink)
	e.GET("/job/", ProcessRequestID)

	return e
}

func CreateEchoWithServer(ctx context.Context, config *ServerConfig) (*echo.Echo, *http.Server) {
	logger := zerolog.Ctx(ctx)

	listenAddr := fmt.Sprintf("%v:%d", config.AddressRedis, config.PortRedis)
	asynqClient := asynq.NewClient(&asynq.RedisClientOpt{
		Addr:     listenAddr,
		Password: config.RedisPw,
		DB:       0,
	})

	asynqInspector := asynq.NewInspector(&asynq.RedisClientOpt{
		Addr:     listenAddr,
		Password: config.RedisPw,
		DB:       0,
	})

	e := createEcho(config, logger.With().Logger(), asynqClient, asynqInspector)

	listenAddr = fmt.Sprintf(":%d", config.Port)

	srv := &http.Server{
		Addr:        listenAddr,
		Handler:     e,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}

	return e, srv
}
