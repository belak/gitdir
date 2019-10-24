package main

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
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

func yamlLookupKey(n *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Kind == yaml.ScalarNode && n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}

	return nil
}

func ensureAdmin(targetNode *yaml.Node, val bool) {
	// We only want to set the value if it's true
	if !val {
		return
	}

	adminValue := yamlLookupKey(targetNode, "is_admin")

	newNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!bool",
		Value: "true",
	}

	if adminValue == nil {
		targetNode.Content = append(
			targetNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "is_admin",
			},
			newNode,
		)
	} else {
		*adminValue = *newNode
	}
}

func ensureKey(targetNode *yaml.Node, val *publicKey) {
	keysValue := yamlLookupKey(targetNode, "keys")

	if keysValue == nil {
		keysValue = &yaml.Node{
			Kind: yaml.SequenceNode,
		}
		targetNode.Content = append(
			targetNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "keys",
			},
			keysValue,
		)
	}

	keysValue.Content = append(keysValue.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: val.MarshalAuthorizedKey(),
	})
}

func cmdAddUser(c *cli.Context) error {
	config, err := NewCLIConfig(c)
	if err != nil {
		return err
	}

	username := c.String("username")
	pubkey := c.Generic("pubkey").(*publicKey)
	admin := c.Bool("admin")

	repo, worktreeFS, err := config.EnsureRepo("admin/admin")
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Force: true,
	})
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return err
	}

	filename := "users/" + username + ".yml"

	data, err := readFile(worktreeFS, filename)
	if err != nil {
		log.Warn().Str("filename", filename).Err(err).Msg("File missing")
	}

	rootNode := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	// We explicitly ignore this error so we can manually make a tree
	_ = yaml.Unmarshal(data, rootNode)

	if len(rootNode.Content) == 0 {
		rootNode.Content = append(rootNode.Content, &yaml.Node{
			Kind: yaml.MappingNode,
		})
	}

	if len(rootNode.Content) != 1 || rootNode.Content[0].Kind != yaml.MappingNode {
		return errors.New("Root is not a valid yaml document")
	}

	targetNode := rootNode.Content[0]

	ensureAdmin(targetNode, admin)
	ensureKey(targetNode, pubkey)

	out, err := yaml.Marshal(rootNode)
	if err != nil {
		return err
	}

	f, err := worktreeFS.Create(filename)
	if err != nil {
		return err
	}

	_, err = f.Write(out)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	_, err = worktree.Add(filename)
	if err != nil {
		return err
	}

	_, err = worktree.Commit("Added key to "+username, &git.CommitOptions{
		Author: newAdminGitSignature(),
	})
	if err != nil {
		return err
	}

	log.Info().Msg("Success!")
	return nil
}
