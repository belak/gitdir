package main

import (
	"github.com/rs/zerolog/log"

	"github.com/belak/go-gitdir"
)

func cmdServe(c Config) {
	log.Info().Msg("starting server")

	serv, err := gitdir.NewServer(c.FS())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load SSH server")
	}

	serv.Addr = c.BindAddr

	err = serv.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run SSH server")
	}
}
