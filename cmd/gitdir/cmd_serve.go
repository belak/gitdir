package main

import (
	"github.com/rs/zerolog/log"

	"github.com/belak/go-gitdir"
)

func cmdServe(c Config) {
	log.Info().Msg("starting server")

	serv, err := gitdir.NewServer(c.FS())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load SSH server")
	}

	// If both the AdminUser and AdminPublicKey were set, attempt to add that
	// user to the config.
	if c.AdminUser != "" && c.AdminPublicKey != nil {
		log.Info().Str("user", c.AdminUser).Msg("ensuring admin user exists")

		err = serv.EnsureAdminUser(c.AdminUser, c.AdminPublicKey)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to add admin user")
		}
	}

	serv.Addr = c.BindAddr

	err = serv.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run SSH server")
	}
}
