package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"ripper-api/server"

	"github.com/urfave/cli/v2"
)

func Run(cCtx *cli.Context) error {
	serverConfig, err := initConfig(cCtx)

	if err != nil {
		return err
	}

	logger := initLogger()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	ctx = logger.WithContext(ctx)
	defer stop()

	e, srv := server.CreateEchoWithServer(
		logger.With().Str("component", "server").Logger().WithContext(ctx),
		serverConfig,
	)

	// start the http server
	go func() {
		if err := e.StartServer(srv); err != nil && err != http.ErrServerClosed {
			logger.Fatal().
				AnErr("error", err).
				Msg("Error starting HTTP listener")
		}
	}()

	<-ctx.Done()

	logger.Info().Msg("Attempting graceful shutdown, Ctrl+C to force")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx = logger.WithContext(ctx)
	defer cancel()

	// trigger echo graceful shutdown
	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal().
			AnErr("error", err).
			Msg("Error while shutting down")
	}

	return nil
}

func Start() {
	app := cli.App{
		Name:        "ripper-api",
		Usage:       "Web server for amusic ripping",
		Description: "Web server with alac ripping, coverting and removing padding. Works with frida server and amusic wrapper",
		UsageText:   "ripper-api [flags]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "bind-addr",
				Usage:   "Server listen address",
				Value:   "127.0.0.1:8100",
				EnvVars: []string{"BIND_ADDR"},
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:    "web-dir",
				Usage:   "Temporary directory for content serving",
				EnvVars: []string{"WEB_DIR"},
				Aliases: []string{"d"},
			},
			&cli.StringFlag{
				Name:    "bind-wrapper",
				Usage:   "Address and port wrapper listens on",
				EnvVars: []string{"BIND_ADDR_WRAPPER"},
				Aliases: []string{"w"},
			},
			&cli.StringFlag{
				Name:    "key-db",
				Usage:   "File with valid api keys",
				Value:   "./keys",
				EnvVars: []string{"KEY_DB"},
				Aliases: []string{"k"},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
