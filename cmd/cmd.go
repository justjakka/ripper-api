package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

	wrappers := cCtx.StringSlice("wrappers")

	queues := make(map[string]int)

	for i := range wrappers {
		queuename := strconv.Itoa(i)
		queues[queuename] = 3
	}

	qsrv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cCtx.String("redis"),
			Password: cCtx.String("redis-pw"),
			DB:       0,
		},
		asynq.Config{
			Concurrency: 2,
			Queues:      queues,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(ripper.TypeRip, ripper.HandleProcessTask)
	mux.HandleFunc(ripper.TypeInit, ripper.HandleInitQueueTask)

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
						Name:     "web-dir",
						Usage:    "Temporary directory for content serving",
						EnvVars:  []string{"WEB_DIR"},
						Aliases:  []string{"d"},
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:     "wrappers",
						Usage:    "Wrapper addresses and ports",
						EnvVars:  []string{"WRAPPERS"},
						Aliases:  []string{"w"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "key-db",
						Usage:    "File with valid api keys",
						EnvVars:  []string{"KEY_DB"},
						Aliases:  []string{"k"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "redis",
						Usage:    "Address and port of redis",
						EnvVars:  []string{"REDIS_ADDRESS"},
						Aliases:  []string{"r"},
						Required: true,
					},
					&cli.StringFlag{
						Name:    "redis-pw",
						Usage:   "Redis DB password",
						Value:   "",
						EnvVars: []string{"REDIS_PASSWORD"},
						Aliases: []string{"pw"},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
