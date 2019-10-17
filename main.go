package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load config first so we know how to set the logger
	c, err := NewEnvConfig()

	// Set up the logger
	if c.LogReadable {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	}
	if c.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Msg("Starting go-git-dir")

	if err != nil {
		log.Fatal().Err(err).Msg("Error loading environment config")
	}

	serv, err := newServer(NewDefaultConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load SSH server")
	}

	err = serv.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run SSH server")
	}
}
