package main

import (
	"github.com/rs/zerolog/log"

	"github.com/belak/go-gitdir"
)

func cmdServe(c gitdir.Config) {
	log.Info().Msg("starting server")

	serv, err := gitdir.NewServer(&c)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load SSH server")
	}

	err = serv.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run SSH server")
	}
}
