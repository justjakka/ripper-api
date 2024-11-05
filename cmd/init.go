package cmd

import (
	"bufio"
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"time"

	"ripper-api/server"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
)

type Config struct {
	Port         uint     `toml:"Port"`
	Address	     string   `toml:"Address"`
	AddressRedis string   `toml:"Redis"`
	Wrappers     []string `toml:"Wrappers"`
	WebDir       string   `toml:"Webdir"`
	RedisPw      string   `toml:"RedisPw"`
	Keyfile      string   `toml:"Keyfile"`
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Failed to close file")
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func initConfig(cCtx *cli.Context) (*server.Config, error) {
	var conf Config
	if cCtx.Path("config") != "" {
		if _, err := toml.DecodeFile(cCtx.Path("config"), &conf); err != nil {
			return nil, err
		}
		lines, err := readLines(conf.Keyfile)
		if err != nil {
			return nil, err
		}

		return &server.Config{
				Port:         conf.Port,
				Address:      conf.Address,
				Wrappers:     conf.Wrappers,
				WebDir:       conf.WebDir,
				RedisPw:      conf.RedisPw,
				AddressRedis: conf.AddressRedis,
				KeyList:      lines},
			nil
	} else {
		lines, err := readLines(cCtx.String("key-db"))
		if err != nil {
			return nil, err
		}

		wrappers := cCtx.StringSlice("wrappers")

		return &server.Config{
				Port:         cCtx.Uint("port"),
				Address:      cCtx.String("address"),
				Wrappers:     wrappers,
				WebDir:       cCtx.String("web-dir"),
				RedisPw:      cCtx.String("redis-pw"),
				AddressRedis: cCtx.String("redis"),
				KeyList:      lines},
			nil
	}
}

func initLogger() zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	logger := zerolog.New(output).With().Timestamp().Logger()
	log.SetFlags(0)
	log.SetOutput(logger)

	return logger
}
