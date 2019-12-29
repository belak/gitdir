package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Look for problems preventing startup",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_, err := NewEnvConfig()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load base config")
			}

			fmt.Println(args)
		},
	}

	// flags := cmd.Flags()

	return cmd
}
