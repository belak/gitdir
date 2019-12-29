package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/belak/go-gitdir"
	"github.com/belak/go-gitdir/models"
)

func hookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := NewEnvConfig()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load base config")
			}

			log.Info().Msg("starting hook")

			if len(os.Args) < 3 {
				log.Fatal().Msg("missing hook name")
			}

			path, ok := os.LookupEnv("GITDIR_HOOK_REPO_PATH")
			if !ok {
				log.Fatal().Msg("missing repo path")
			}

			pkData, ok := os.LookupEnv("GITDIR_HOOK_PUBLIC_KEY")
			if !ok {
				log.Fatal().Msg("missing public key")
			}

			pk, err := models.ParsePublicKey([]byte(pkData))
			if err != nil {
				log.Fatal().Err(err).Msg("failed to parse public key")
			}

			config := gitdir.NewConfig(c.FS())

			err = config.Load()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load gitdir")
			}

			// Call the actual hook
			err = config.RunHook(os.Args[2], path, pk, os.Args[3:], os.Stdin)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	return cmd
}
