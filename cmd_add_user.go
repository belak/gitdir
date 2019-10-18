package main

import (
	"errors"
	"path/filepath"

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
			Value:    &publicKey{},
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
	pubkey := c.Generic("pubkey").(*publicKey)
	admin := c.Bool("admin")

	repo, err := EnsureRepo(filepath.Join(config.BasePath, "admin", "admin"))
	if err != nil {
		return err
	}

	builder := repo.CommitBuilder()

	err = builder.UpdateFile("users/"+username+".yml", func(data []byte) ([]byte, error) {
		rootNode := &yaml.Node{}

		// We explicitly ignore this error so we can manually make a tree
		_ = yaml.Unmarshal(data, rootNode)

		if rootNode == nil {
			rootNode = &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{{
					Kind: yaml.MappingNode,
				}},
			}
		}

		if len(rootNode.Content) != 1 || rootNode.Content[0].Kind != yaml.MappingNode {
			return nil, errors.New("Root is not a valid yaml document")
		}

		targetNode := rootNode.Content[0]

		if admin {
			var adminNode *yaml.Node
			var adminNodeValue *yaml.Node
			for i := 0; i+1 < len(targetNode.Content); i += 2 {
				if targetNode.Content[i].Kind == yaml.ScalarNode && targetNode.Content[i].Value == "is_admin" {
					adminNode = targetNode.Content[i]
					adminNodeValue = targetNode.Content[i+1]
				}
			}

			if adminNode == nil {
				adminNode = &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "is_admin",
				}
				adminNodeValue = &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!bool",
					Value: "true",
				}
				targetNode.Content = append(
					targetNode.Content,
					adminNode,
					adminNodeValue,
				)
			} else {
				*adminNodeValue = yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!bool",
					Value: "true",
				}
			}
		}

		var keysNode *yaml.Node
		var keysNodeValue *yaml.Node
		for i := 0; i+1 < len(targetNode.Content); i += 2 {
			if targetNode.Content[i].Kind == yaml.ScalarNode && targetNode.Content[i].Value == "keys" {
				keysNode = targetNode.Content[i]
				keysNodeValue = targetNode.Content[i+1]
			}
		}

		if keysNode == nil {
			keysNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "keys",
			}
			keysNodeValue = &yaml.Node{
				Kind: yaml.SequenceNode,
			}
			targetNode.Content = append(targetNode.Content, keysNode, keysNodeValue)
		}

		targetNode = keysNodeValue
		targetNode.Content = append(targetNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.SingleQuotedStyle,
			Value: pubkey.String(),
		})

		return yaml.Marshal(rootNode)
	})
	if err != nil {
		return err
	}

	_, err = builder.Write("Added key to "+username, nil, nil)
	if err != nil {
		return err
	}

	log.Info().Msg("Success!")
	return nil
}
