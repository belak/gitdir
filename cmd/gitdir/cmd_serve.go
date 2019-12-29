package main

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/belak/go-gitdir"
)

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the git SSH server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := NewEnvConfig()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load base config")
			}

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
		},
	}

	return cmd
}
