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
	BindAddr   string
	GitUser    string
	BasePath   string
	UserPrefix string
	OrgPrefix  string

	LogReadable bool
	LogDebug    bool
}

func cliFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			EnvVar: "GITDIR_DEBUG",
			Usage:  "Enable debug logging",
		},
		cli.BoolFlag{
			Name:   "log-readable",
			EnvVar: "GITDIR_LOG_READABLE",
			Usage:  "Enable human readable logging",
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
			Value:  ":2222",
			Usage:  "Host and port to bind to",
		},

		// TODO: move the following to the admin config
		cli.StringFlag{
			Name:   "user-prefix",
			EnvVar: "GITDIR_USER_PREFIX",
			Value:  "~",
			Usage:  "Prefix to use when cloning user repos",
		},
		cli.StringFlag{
			Name:   "org-prefix",
			EnvVar: "GITDIR_ORG_PREFIX",
			Value:  "@",
			Usage:  "Prefix to use when cloning org repos",
		},
	}
}

func NewCLIConfig(ctx *cli.Context) (*Config, error) {
	c := NewDefaultConfig()

	c.LogReadable = ctx.GlobalBool("log-readable")
	c.LogDebug = ctx.GlobalBool("debug")
	c.BasePath = ctx.GlobalString("base-dir")
	c.BindAddr = ctx.GlobalString("bind-addr")
	c.UserPrefix = ctx.GlobalString("user-prefix")
	c.OrgPrefix = ctx.GlobalString("org-prefix")

	// Set up the logger
	if c.LogReadable {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	}
	if c.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	err := os.Chdir(c.BasePath)
	if err != nil {
		return c, err
	}

	return c, nil
}

// NewDefaultConfig returns the base config.
func NewDefaultConfig() *Config {
	return &Config{
		BindAddr:   ":2222",
		GitUser:    "git",
		BasePath:   "./tmp",
		UserPrefix: "~",
		OrgPrefix:  "@",
	}
}
