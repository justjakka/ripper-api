package cmd

import (
	"context"
	"errors"
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
	err = os.MkdirAll(serverConfig.WebDir, os.ModePerm)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	logger := initLogger()

	queues := make(map[string]int)

	for i := range len(serverConfig.Wrappers) {
		queues[fmt.Sprintf("%v", i)] = 3
	}

	qsrv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     serverConfig.AddressRedis,
			Password: serverConfig.RedisPw,
			DB:       0,
		},
		asynq.Config{
			Concurrency: len(serverConfig.Wrappers),
			Queues:      queues,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(ripper.TypeRip, ripper.HandleProcessTask)
	mux.HandleFunc(ripper.TypeInit, ripper.HandleInitQueueTask)
	mux.HandleFunc(ripper.TypeDelete, ripper.HandleDeleteTask)

	// start asynq server
	go func() {
		if err := qsrv.Run(mux); err != nil {
			logger.Fatal().
				AnErr("error", err).
				Msg("Error starting Asynq server")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	ctx = logger.WithContext(ctx)
	defer stop()

	e, srv := server.CreateEchoWithServer(
		logger.With().Str("component", "server").Logger().WithContext(ctx),
		serverConfig,
	)

	// start the http server
	go func() {
		if err := e.StartServer(srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	qsrv.Stop()
	qsrv.Shutdown()
	logger.Info().Msg("Removing everything from web folder")
	err = os.RemoveAll(serverConfig.WebDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(serverConfig.WebDir, os.ModePerm)
	if err != nil {
		return err
	}

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
				Usage:  "Run the HTTP/asynq server",
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
						Name:	 "address",
						Usage:	 "Address to bind the HTTP listener to",
						Value:	 string("127.0.0.1"),
						EnvVars: []string{"ADDRESS"},
						Aliases: []string{"a"},
					},
					&cli.StringFlag{
						Name:    "web-dir",
						Usage:   "Temporary directory for content serving",
						EnvVars: []string{"WEB_DIR"},
						Aliases: []string{"d"},
					},
					&cli.StringSliceFlag{
						Name:    "wrappers",
						Usage:   "Wrapper addresses and ports",
						EnvVars: []string{"WRAPPERS"},
						Aliases: []string{"w"},
					},
					&cli.StringFlag{
						Name:    "key-db",
						Usage:   "File with valid api keys",
						EnvVars: []string{"KEY_DB"},
						Aliases: []string{"k"},
					},
					&cli.StringFlag{
						Name:    "redis",
						Usage:   "Address and port of redis",
						EnvVars: []string{"REDIS_ADDRESS"},
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:    "redis-pw",
						Usage:   "Redis DB password",
						Value:   "",
						EnvVars: []string{"REDIS_PASSWORD"},
						Aliases: []string{"pw"},
					},
					&cli.PathFlag{
						Name:    "config",
						Usage:   "Path for config file",
						Value:   "",
						EnvVars: []string{"RIPPER_CONFIG"},
						Aliases: []string{"c"},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
