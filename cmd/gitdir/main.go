package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

func main() {
	c, err := NewEnvConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load base config")
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "hook" {
			log.Fatal().Msg("sub-command not found")
		}

		cmdHook(c)

		return
	}

	log.Info().Msg("starting go-gitdir")

	cmdServe(c)
}
