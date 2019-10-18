package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Flags = cliFlags()
	app.Commands = []cli.Command{
		{
			Name:   "serve",
			Usage:  "run the server",
			Action: cmdServe,
		},
		{
			Name:   "add-user",
			Usage:  "add a user to the config repo",
			Action: cmdAddUser,
			Flags:  addUserFlags(),
		},
	}

	// We actually want to default to the serve command rather than help.
	app.Action = cmdServe

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("Command failed")
	}
}
