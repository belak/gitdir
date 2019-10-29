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

// TODO: clean this up
func cmdAddUser(c *cli.Context) error { //nolint:funlen
	// Load the CLI config - note that this will switch to the proper basedir
	_, err := NewCLIConfig(c)
	if err != nil {
		return err
	}

	username := c.String("username")
	pubkey := c.Generic("pubkey").(*PublicKey)
	admin := c.Bool("admin")

	adminRepo, err := EnsureRepo("admin/admin", true)
	if err != nil {
		return err
	}

	// Make sure the user repo exists, just in case, even if we don't plan on
	// using it.
	_, err = EnsureRepo("admin/user-"+username, true)
	if err != nil {
		return err
	}

	err = adminRepo.UpdateFile("config.yml", func(data []byte) ([]byte, error) {
		rootNode, _, err := ensureSampleConfigYaml(data) //nolint:govet
		if err != nil {
			return nil, err
		}

		// We can assume the config file is in a valid format because of
		// ensureSampleConfig
		targetNode := rootNode.Content[0]
		usersVal := yamlLookupVal(targetNode, "users")
		userVal, _ := yamlEnsureKey(usersVal, username, &yaml.Node{Kind: yaml.MappingNode}, "", false)

		keysVal, _ := yamlEnsureKey(userVal, "keys", &yaml.Node{Kind: yaml.SequenceNode}, "", false)
		keysVal.Content = append(keysVal.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: pubkey.String(),
		})

		if admin {
			yamlEnsureKey(
				userVal,
				"is_admin",
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!bool",
					Value: "true",
				},
				"",
				true,
			)
		}

		return yamlEncode(rootNode)
	})
	if err != nil {
		return err
	}

	err = adminRepo.Commit("Updated user "+username, nil)
	if err != nil {
		return err
	}

	log.Info().Msg("Success!")

	return nil
}
