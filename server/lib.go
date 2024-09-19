package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func createEcho(config *ServerConfig, logger zerolog.Logger) *echo.Echo {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &ConfigContext{c, config}
			return next(cc)
		}
	})

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Str("method", v.Method).
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

	e.POST("/:urlhex", ProcessLink)
	e.GET("/requestid/:reqid", ProcessRequestID)

	/* var JobQueue = make(map[string]Job) */

	return e
}

func CreateEchoWithServer(ctx context.Context, config *ServerConfig) (*echo.Echo, *http.Server) {
	logger := zerolog.Ctx(ctx)

	e := createEcho(config, logger.With().Logger())
	listenAddr := fmt.Sprintf(":%d", config.Port)

	srv := &http.Server{
		Addr:        listenAddr,
		Handler:     e,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}

	return e, srv
}
