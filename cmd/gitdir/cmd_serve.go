package main

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/belak/go-gitdir"
)

func LoadServer() (*gitdir.Server, error) {
	c, err := NewEnvConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load base config")
	}

	serv, err := gitdir.NewServer(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load server config")
	}

	return serv, nil
}

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the git SSH server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			serv, err := LoadServer()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load server")
			}

			err = serv.ListenAndServe()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to run SSH server")
			}
		},
	}

	return cmd
}
