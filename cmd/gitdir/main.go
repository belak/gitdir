package main

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gitdir",
		Short: "gitdir is a simple git SSH server.",
		Args:  cobra.NoArgs,
	}

	// Add all the subcommands
	rootCmd.AddCommand(
		hookCmd(),
		initCmd(),
		doctorCmd(),
		serveCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run command")
	}
}
