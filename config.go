package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

// There are two places things can be configured. Environment variables are
// meant to handle deployment level config (like where the repos are placed) and
// user configuration happens in a repo called admin. There are two levels of
// admin repos. One at the instance level to define site settings and one at the
// org level.
//
// This file is meant for environment variable settings.

type Config struct {
	BindAddr  string
	BasePath  string
	LogFormat string
	LogDebug  bool
}

var DefaultConfig = &Config{
	BindAddr:  ":2222",
	BasePath:  "./tmp",
	LogFormat: "json",
}

func cliFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			EnvVar: "GITDIR_DEBUG",
			Usage:  "Enable debug logging",
		},
		cli.StringFlag{
			Name:   "log-format",
			EnvVar: "GITDIR_LOG_FORMAT",
			Usage:  "Log format: console or json",
			Value:  DefaultConfig.LogFormat,
		},
		cli.StringFlag{
			Name:     "base-dir",
			EnvVar:   "GITDIR_BASE_DIR",
			Required: true,
			Usage:    "Which directory to operate on",
		},
		cli.StringFlag{
			Name:   "bind-addr",
			EnvVar: "GITDIR_BIND_ADDR",
			Value:  DefaultConfig.BindAddr,
			Usage:  "Host and port to bind to",
		},
	}
}

func NewCLIConfig(ctx *cli.Context) (Config, error) {
	c := *DefaultConfig

	c.LogFormat = ctx.GlobalString("log-format")
	c.LogDebug = ctx.GlobalBool("debug")
	c.BasePath = ctx.GlobalString("base-dir")
	c.BindAddr = ctx.GlobalString("bind-addr")

	// Set up the logger - anything other than console defaults to json.
	if c.LogFormat == "console" {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	}

	if c.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// It's easier to handle repo operations down the line if we switch to the
	// base path. Also, repo names are prettier when you run into errors.
	err := os.Chdir(c.BasePath)
	if err != nil {
		return c, err
	}

	return c, nil
}
