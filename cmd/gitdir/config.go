package main

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

// Config stores all the server-level settings. These cannot be changed at
// runtime. They are only used by the binary and are passed to the proper
// places.
type Config struct {
	BindAddr  string
	BasePath  string
	LogFormat string
	LogDebug  bool
}

// FS returns the billy.Filesystem for this base path.
func (c Config) FS() billy.Filesystem {
	return osfs.New(c.BasePath)
}

// DefaultConfig is used as the base config.
var DefaultConfig = Config{
	BindAddr:  ":2222",
	BasePath:  "./tmp",
	LogFormat: "json",
}

// NewEnvConfig returns a new Config based on environment variables.
func NewEnvConfig() (Config, error) {
	var err error

	c := DefaultConfig

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

	info, err := os.Stat(c.BasePath)
	if err != nil {
		return c, errors.Wrap(err, "GITDIR_BASE_DIR")
	}

	if !info.IsDir() {
		return c, errors.New("GITDIR_BASE_DIR: not a directory")
	}

	return c, nil
}
