package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	c, err := NewEnvConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load base config")
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "hook":
			cmdHook(c)
		default:
			log.Fatal().Msg("sub-command not found")
		}

		return
	}

	log.Info().Msg("starting go-gitdir")

	cmdServe(c)
}
