package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

func main() {
	c, err := NewEnvConfig()

	log.Info().Msg("starting go-gitdir")

	if err != nil {
		log.Fatal().Err(err).Msg("failed to load base config")
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "hook" {
			log.Fatal().Msg("sub-command not found")
		}

		// TODO: call hook
		log.Fatal().Msg("hook not implemented")
	}

	cmdServe(c)
}
