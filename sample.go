package main

import (
	yaml "gopkg.in/yaml.v3"
)

func ensureSampleConfig(data []byte) ([]byte, error) {
	rootNode, modified, err := ensureSampleConfigYaml(data)
	if err != nil {
		return nil, err
	}

	if !modified {
		return data, nil
	}

	return yamlEncode(rootNode)
}

func ensureSampleConfigYaml(data []byte) (*yaml.Node, bool, error) {
	rootNode, targetNode, err := yamlEnsureDocument(data)
	if err != nil {
		return nil, false, err
	}

	vals := [4]bool{
		ensureSampleUsers(targetNode),
		ensureSampleGroups(targetNode),
		ensureSampleOrgs(targetNode),
		ensureSampleOptions(targetNode),
	}

	// If we had to make any of the modifications, we need to specify the node
	// was updated.
	if vals[0] || vals[1] || vals[2] || vals[3] {
		return rootNode, true, nil
	}

	return rootNode, false, nil
}

func ensureSampleUsers(targetNode *yaml.Node) bool {
	_, modified := yamlEnsureKey(
		targetNode,
		"users",
		&yaml.Node{Kind: yaml.MappingNode},
		`
Users defines the users who have access to the service. They will need an
SSH key or invite added to their user account before they can access the
server.
#
Sample user (with all options set):
#
belak:
  is_admin: true
  disabled: false
  keys:
    - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDeQfBUWIqpGXS8xCOg/0RKVOGTnzpIdL7r9wK1/xA52 belak@tmp`,
		false,
	)

	return modified
}

func ensureSampleGroups(targetNode *yaml.Node) bool {
	_, modified := yamlEnsureKey(
		targetNode,
		"groups",
		&yaml.Node{Kind: yaml.MappingNode},
		`
Groups can be used in any place a normal user could be used. They are prefixed
with a $, so the admins group could be accessed with $admins. Groups can be
defined recursively, but they cannot have loops.
#
Sample groups:
#
groups:
  admins:
	- belak
	- some-trusted-user
  vault-members:
	- $admins
	- some-less-trusted-user`,
		false,
	)

	return modified
}

func ensureSampleOrgs(targetNode *yaml.Node) bool {
	_, modified := yamlEnsureKey(
		targetNode,
		"orgs",
		&yaml.Node{Kind: yaml.MappingNode},
		`
Org repos are accessible at @org-name/repo. Note that if admins is not
specified, the only users with admin access will be global admins. By
default, all members of an org will have read-write access to repos. This
can be changed with the read and write keys.
#
Sample org (with all options set):
#
vault:
  admins:
	- belak
  write:
    - some user
  read:
	- some-other-user
  repos:
	project-name-here:
	  public: false
	  write:
	    - belak
	  read:
		- some-user
		- some-other-user`,
		false,
	)

	return modified
}

// NOTE: this would make more sense as a map, but we want to keep the order.
var sampleOptions = []struct {
	Name    string
	Comment string
	Tag     string
	Value   string
}{
	{
		Name:    "git_user",
		Comment: "which username to use as the global git user",
		Value:   defaultAdminOptions.GitUser,
	},
	{
		Name:    "org_prefix",
		Comment: "the prefix to use when cloning org repos",
		Value:   defaultAdminOptions.OrgPrefix,
	},
	{
		Name:    "user_prefix",
		Comment: "the prefix to use when cloning user repos",
		Value:   defaultAdminOptions.UserPrefix,
	},
	{
		Name: "implicit_repos",
		Comment: `allow users with admin access to a given area to create repos by simply
pushing to them.`,
		Tag:   "!!bool",
		Value: "false",
	},
	{
		Name: "user_config_keys",
		Comment: `allows users to specify ssh keys in their own config, rather than relying
on the main admin config.`,
		Tag:   "!!bool",
		Value: "false",
	},
	{
		Name: "user_config_repos",
		Comment: `allows users to specify repos in their own config, rather than relying on
the main admin config.`,
		Tag:   "!!bool",
		Value: "false",
	},
	{
		Name: "org_config_repos",
		Comment: `allows org admins to specify repos in their own config, rather than
relying on the main admin config.`,
		Tag:   "!!bool",
		Value: "false",
	},
}

func ensureSampleOptions(targetNode *yaml.Node) bool {
	optionsVal, modified := yamlEnsureKey(
		targetNode,
		"options",
		&yaml.Node{Kind: yaml.MappingNode},
		"",
		false,
	)

	// Ensure all our options are on the options struct.
	for _, opt := range sampleOptions {
		_, added := yamlEnsureKey(
			optionsVal,
			opt.Name,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   opt.Tag,
				Value: opt.Value,
			},
			opt.Comment,
			false,
		)

		modified = modified || added
	}

	return modified
}
