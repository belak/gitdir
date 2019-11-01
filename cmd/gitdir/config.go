package main

import (
	"os"
	"strconv"

	"github.com/belak/go-gitdir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// NewEnvConfig returns a new gitdir.Config based on environment variables.
func NewEnvConfig() (gitdir.Config, error) {
	var err error

	c := *gitdir.DefaultConfig

	if rawDebug, ok := os.LookupEnv("GITDIR_DEBUG"); ok {
		c.LogDebug, err = strconv.ParseBool(rawDebug)
		if err != nil {
			return c, errors.Wrap(err, "GITDIR_DEBUG")
		}
	}

	if logFormat, ok := os.LookupEnv("GITDIR_LOG_FORMAT"); ok {
		if logFormat != "console" && logFormat != "json" {
			return c, errors.New("GITDIR_LOG_FORMAT: must be console or json")
		}
	}

	// Set up the logger - anything other than console defaults to json.
	if c.LogFormat == "console" {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	}

	if c.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if bindAddr, ok := os.LookupEnv("GITDIR_BIND_ADDR"); ok {
		c.BindAddr = bindAddr
	}

	var ok bool

	if c.BasePath, ok = os.LookupEnv("GITDIR_BASE_DIR"); !ok {
		return c, errors.New("GITDIR_BASE_DIR: not set")
	}

	// It's easier to handle repo operations down the line if we switch to the
	// base path. Also, repo names are prettier when you run into errors.
	//
	// TODO: this should be moved into the gitdir package so you can have
	// multiple gitdirs at once.
	err = os.Chdir(c.BasePath)
	if err != nil {
		return c, errors.Wrap(err, "GITDIR_BASE_DIR")
	}

	return c, nil
}
