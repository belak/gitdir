package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func cmdServe(c *cli.Context) error {
	// Load config first so we know how to set the logger
	config, err := NewCLIConfig(c)

	// Set up the logger
	if config.LogReadable {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	}
	if config.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Msg("Starting go-git-dir")

	if err != nil {
		log.Fatal().Err(err).Msg("Error loading environment config")
	}

	serv, err := newServer(config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load SSH server")
	}

	err = serv.ListenAndServe()
	if err != nil {
		// We use our own Fatal call here rather than falling back to main so we
		// can match the log format.
		log.Fatal().Err(err).Msg("Failed to run SSH server")
	}

	return nil
}
