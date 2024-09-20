package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"ripper-api/ripper"
	"ripper-api/server"

	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"
)

func serve(cCtx *cli.Context) error {
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

	listenAddr := fmt.Sprintf("%v:%d", cCtx.String("redis-address"), cCtx.Uint("port-redis"))
	qsrv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     listenAddr,
			Password: cCtx.String("redis-pw"),
			DB:       0,
		},
		asynq.Config{
			Concurrency: 1,
			Queues: map[string]int{
				"default": 3,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(ripper.TypeRip, ripper.HandleRipTask)
	// start asynq server
	go func() {
		if err := qsrv.Run(mux); err != nil {
			logger.Fatal().
				AnErr("error", err).
				Msg("Error starting Asynq server")
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

	qsrv.Shutdown()

	return nil
}

func Start() {
	app := cli.App{
		Name:        "ripper-api",
		Usage:       "Web server for amusic ripping",
		Description: "Web server with alac ripping, coverting and removing padding. Works with frida server and amusic wrapper",
		UsageText:   "ripper-api [flags]",
		Commands: []*cli.Command{
			{Name: "serve",
				Usage:  "Run the HTTP server",
				Action: serve,
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:    "port",
						Usage:   "Port to bind the HTTP listener to",
						Value:   uint(8080),
						EnvVars: []string{"PORT"},
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:    "web-dir",
						Usage:   "Temporary directory for content serving",
						EnvVars: []string{"WEB_DIR"},
						Aliases: []string{"d"},
					},
					&cli.StringFlag{
						Name:    "port-wrapper",
						Usage:   "Port wrapper listens on",
						EnvVars: []string{"WRAPPER_PORT"},
						Aliases: []string{"w"},
					},
					&cli.StringFlag{
						Name:    "key-db",
						Usage:   "File with valid api keys",
						Value:   "./keys",
						EnvVars: []string{"KEY_DB"},
						Aliases: []string{"k"},
					},
					&cli.StringFlag{
						Name:    "port-redis",
						Usage:   "Port redis listens on",
						EnvVars: []string{"REDIS_PORT"},
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:    "redis-pw",
						Usage:   "Redis password",
						EnvVars: []string{"REDIS_PASSWORD"},
						Aliases: []string{"pw"},
					},
					&cli.StringFlag{
						Name:    "redis-address",
						Usage:   "Redis address",
						EnvVars: []string{"REDIS_ADDRESS"},
						Aliases: []string{"ad"},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
