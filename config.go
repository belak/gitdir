package main

import (
	"errors"
	"os"
)

// There are two places things can be configured. Environment variables are
// meant to handle deployment level config (like where the repos are placed) and
// user configuration happens in a repo called admin. There are two levels of
// admin repos. One at the instance level to define site settings and one at the
// org level.
//
// This file is meant for environment variable settings.

type Config struct {
	Addr       string
	GitUser    string
	BasePath   string
	UserPrefix string
	OrgPrefix  string

	LogReadable bool
	LogDebug    bool
}

// NewEnvConfig will return a new config object and whether or not the config is
// valid. Note that this will always return a config object even if it also
// returns an error so logging can be configured properly.
func NewEnvConfig() (*Config, error) {
	c := NewDefaultConfig()
	c.LogReadable = getenvBool("GITDIR_LOG_READABLE", false)
	c.LogDebug = getenvBool("GITDIR_DEBUG", false)

	dir, ok := os.LookupEnv("GITDIR_BASE_DIR")
	if !ok {
		return c, errors.New("No GITDIR_BASE_DIR set")
	}

	if info, err := os.Stat(dir); os.IsNotExist(err) {
		return c, errors.New("GITDIR_BASE_DIR does not exist")
	} else if !info.IsDir() {
		return c, errors.New("GITDIR_BASE_DIR is not a directory")
	}

	return c, nil
}

// NewDefaultConfig returns the base config.
func NewDefaultConfig() *Config {
	return &Config{
		Addr:       ":2222",
		GitUser:    "git",
		BasePath:   "./tmp",
		UserPrefix: "~",
		OrgPrefix:  "@",
	}
}
