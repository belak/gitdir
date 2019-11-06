package models

import (
	yaml "gopkg.in/yaml.v3"
)

// AdminConfig is the config.yml that comes from the admin repo.
type AdminConfig struct {
	Invites map[string]string           `yaml:"invites"`
	Users   map[string]*AdminConfigUser `yaml:"users"`
	Orgs    map[string]*OrgConfig       `yaml:"orgs"`
	Repos   map[string]*RepoConfig      `yaml:"repos"`
	Groups  map[string][]string         `yaml:"groups"`
	Options AdminConfigOptions          `yaml:"options"`
}

// AdminConfigUser defines additional fields which main be loaded from the admin
// config.
type AdminConfigUser struct {
	UserConfig `yaml:",inline"`

	IsAdmin  bool `yaml:"is_admin"`
	Disabled bool `yaml:"disabled"`
	// Invites  []string `yaml:"invites"`
}

// NewAdminConfigUser returns a blank AdminConfigUser.
func NewAdminConfigUser() *AdminConfigUser {
	return &AdminConfigUser{
		UserConfig: *NewUserConfig(),
	}
}

// AdminConfigOptions contains all the server level settings which can be
// changed at runtime.
type AdminConfigOptions struct {
	// GitUser refers to which username to use as the global git user.
	GitUser string `yaml:"git_user"`

	// OrgPrefix refers to the prefix to use when cloning org repos.
	OrgPrefix string `yaml:"org_prefix"`

	// UserPrefix refers to the prefix to use when cloning user repos.
	UserPrefix string `yaml:"user_prefix"`

	// InvitePrefix refers to the prefix to use when sshing in with an invite.
	InvitePrefix string `yaml:"invite_prefix"`

	// ImplicitRepos allows a user with admin access to that area to create
	// repos by simply pushing to them.
	ImplicitRepos bool `yaml:"implicit_repos"`

	// UserConfigKeys allows users to specify ssh keys in their own config,
	// rather than relying on the main admin config.
	UserConfigKeys bool `yaml:"user_config_keys"`

	// UserConfigRepos allows users to specify repos in their own config, rather
	// than relying on the main admin config.
	UserConfigRepos bool `yaml:"user_config_repos"`

	// OrgConfig allows org admins to configure orgs in their own config, rather
	// than relying on the main admin config.
	OrgConfig bool `yaml:"org_config"`

	// OrgConfigRepos allows org admins to specify repos in their own config,
	// rather than relying on the main admin config.
	OrgConfigRepos bool `yaml:"org_config_repos"`
}

// DefaultAdminConfigOptions is an object with all values set to their default.
var DefaultAdminConfigOptions = AdminConfigOptions{
	GitUser:      "git",
	OrgPrefix:    "@",
	UserPrefix:   "~",
	InvitePrefix: "invite:",
}

// NewAdminConfig returns a blank admin config with any defaults set.
func NewAdminConfig() *AdminConfig {
	return &AdminConfig{
		Invites: make(map[string]string),
		Users:   make(map[string]*AdminConfigUser),
		Orgs:    make(map[string]*OrgConfig),
		Repos:   make(map[string]*RepoConfig),
		Groups:  make(map[string][]string),

		// Defaults. These should be set in ensure config, but we have them here
		// for reference.
		Options: DefaultAdminConfigOptions,
	}
}

// ParseAdminConfig will return an AdminConfig parsed from the given data. No
// additional validation is done.
func ParseAdminConfig(data []byte) (*AdminConfig, error) {
	ac := NewAdminConfig()

	err := yaml.Unmarshal(data, ac)
	if err != nil {
		return nil, err
	}

	return ac, nil
}
