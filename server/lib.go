package server

import (
	"context"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func createEcho(config *ServerConfig, logger zerolog.Logger) *echo.Echo {
	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")

			return nil
		},
	}))

	e.HideBanner = true

	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())

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

	api := e.Group("/")

	api.POST("/:url", ProcessLink)
	api.GET("/requestid/:reqid", ProcessRequestID)

	return e
}

func CreateEchoWithServer(ctx context.Context, config *ServerConfig) (*echo.Echo, *http.Server) {
	logger := zerolog.Ctx(ctx)

	e := createEcho(config, logger.With().Logger())

	srv := &http.Server{
		Addr:        config.BindAddr,
		Handler:     e,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}

	return e, srv
}
