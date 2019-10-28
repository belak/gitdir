package main

import (
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v3"
)

func addUserFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:     "username",
			Required: true,
			Usage:    "Which username to add or update",
		},
		cli.GenericFlag{
			Name:     "pubkey",
			Required: true,
			Usage:    "File name of a public key to add",
			Value:    &PublicKey{},
		},
		cli.BoolFlag{
			Name:  "admin",
			Usage: "Give the user admin access",
		},
	}
}

func cmdAddUser(c *cli.Context) error {
	config, err := NewCLIConfig(c)
	if err != nil {
		return err
	}

	username := c.String("username")
	pubkey := c.Generic("pubkey").(*PublicKey)
	admin := c.Bool("admin")

	adminRepo, err := config.EnsureRepo("admin/admin", true)
	if err != nil {
		return err
	}

	userRepo, err := config.EnsureRepo("admin/user-"+username, true)
	if err != nil {
		return err
	}

	err = userRepo.UpdateFile("authorized_keys", func(data []byte) ([]byte, error) {
		if len(data) != 0 {
			data = append(data, '\n')
		}

		data = append(data, []byte(pubkey.MarshalAuthorizedKey())...)

		return data, nil
	})
	if err != nil {
		return err
	}

	err = userRepo.Commit("Added key", nil)
	if err != nil {
		return err
	}

	if admin {
		err = adminRepo.UpdateFile("config.yml", func(data []byte) ([]byte, error) {
			rootNode, targetNode, err := yamlEnsureDocument(data)
			if err != nil {
				return nil, err
			}

			// Find the user and add is_admin on
			usersValue := yamlLookupKey(targetNode, "users")
			userValue := yamlLookupKey(usersValue, username)
			yamlEnsureKey(userValue, "is_admin", &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!bool",
				Value: "true",
			})

			return yaml.Marshal(rootNode)
		})
		if err != nil {
			return err
		}

		err = userRepo.Commit("Set "+username+" to admin", nil)
		if err != nil {
			return err
		}
	}

	log.Info().Msg("Success!")
	return nil
}
