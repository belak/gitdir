package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/belak/go-gitdir/models"
)

// Config stores all the server-level settings. These cannot be changed at
// runtime. They are only used by the binary and are passed to the proper
// places.
type Config struct {
	BindAddr       string
	BasePath       string
	LogFormat      string
	LogDebug       bool
	AdminUser      string
	AdminPublicKey *models.PublicKey
}

// FS returns the billy.Filesystem for this base path.
func (c Config) FS() billy.Filesystem {
	return osfs.New(c.BasePath)
}

// DefaultConfig is used as the base config.
var DefaultConfig = Config{
	BindAddr:       ":2222",
	BasePath:       "./tmp",
	LogFormat:      "json",
	LogDebug:       false,
	AdminUser:      "",
	AdminPublicKey: nil,
}

// NewEnvConfig returns a new Config based on environment variables.
func NewEnvConfig() (Config, error) { //nolint:cyclop,funlen
	var err error

	c := DefaultConfig

	if rawDebug, ok := os.LookupEnv("GITDIR_DEBUG"); ok {
		c.LogDebug, err = strconv.ParseBool(rawDebug)
		if err != nil {
			return c, fmt.Errorf("GITDIR_DEBUG: %w", err)
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
		return c, fmt.Errorf("GITDIR_BASE_DIR: not set")
	}

	if c.BasePath, err = filepath.Abs(c.BasePath); err != nil {
		return c, fmt.Errorf("GITDIR_BASE_DIR: %w", err)
	}

	info, err := os.Stat(c.BasePath)
	if err != nil {
		return c, fmt.Errorf("GITDIR_BASE_DIR: %w", err)
	}

	if !info.IsDir() {
		return c, errors.New("GITDIR_BASE_DIR: not a directory")
	}

	// AdminUser and AdminPublicKey are allowed to not be set.
	if adminUser, ok := os.LookupEnv("GITDIR_ADMIN_USER"); ok {
		c.AdminUser = adminUser
	}

	if adminPublicKeyRaw, ok := os.LookupEnv("GITDIR_ADMIN_PUBLIC_KEY"); ok {
		adminPublicKey, err := models.ParsePublicKey([]byte(adminPublicKeyRaw))
		if err != nil {
			return c, fmt.Errorf("GITDIR_ADMIN_PUBLIC_KEY: %w", err)
		}

		c.AdminPublicKey = adminPublicKey
	}

	return c, nil
}
