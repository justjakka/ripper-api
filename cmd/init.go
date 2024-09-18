package cmd

import (
	"bufio"
	"os"
	"time"

	"ripper-api/server"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func initConfig(cCtx *cli.Context) (*server.ServerConfig, error) {
	lines, err := readLines(cCtx.String("key-db"))
	if err != nil {
		return nil, err
	}

	return &server.ServerConfig{
			BindAddr:    cCtx.String("bind-addr"),
			WebDir:      cCtx.String("web-dir"),
			BindWrapper: cCtx.String("bind-wrapper"),
			KeyList:     &lines},
		nil
}

func initLogger() zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	logger := zerolog.New(output).With().Timestamp().Logger()

	return logger
}
