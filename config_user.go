package main

import (
	"github.com/rs/zerolog/log"
)

type UserConfig struct {
	Repos map[string]RepoConfig `yaml:"repos"`
	Keys  []PublicKey           `yaml:"keys"`

	// IsAdmin, Disabled, and Invites can only be specified in admin config
	// files. Note that invites are only valid if the user is set to disabled.
	IsAdmin  bool     `yaml:"is_admin"`
	Disabled bool     `yaml:"disabled"`
	Invites  []string `yaml:"invites"`
}

func loadUser(username string) (UserConfig, error) {
	uc := UserConfig{
		Repos: make(map[string]RepoConfig),
	}

	err := loadConfig("admin/user-"+username, &uc)

	return uc, err
}

func (ac *AdminConfig) loadUserConfigs() {
	for username, user := range ac.Users {
		if user.Repos == nil {
			user.Repos = make(map[string]RepoConfig)
		}

		// Load keys from the admin config
		for _, key := range user.Keys {
			mkey := string(key.Marshal())

			ac.PublicKeys[mkey] = append(ac.PublicKeys[mkey], username)
		}

		// If we have no reason to load config from user repos, we can bail early.
		if !ac.Options.UserConfigRepos && !ac.Options.UserConfigKeys {
			continue
		}

		userConfig, err := loadUser(username)
		if err != nil {
			log.Warn().Err(err).Str("username", username).Msg("Failed to load user repo")
		}

		if ac.Options.UserConfigKeys {
			// Load keys from the user config.
			for _, key := range userConfig.Keys {
				mkey := string(key.Marshal())

				ac.PublicKeys[mkey] = append(ac.PublicKeys[mkey], username)
			}
		}

		if err != nil {
			continue
		}

		if ac.Options.UserConfigRepos {
			// We only really need to merge repos when dealing with loading users,
			// as we don't want them to be able to set config options.
			for repoName, repo := range userConfig.Repos {
				user.Repos[repoName] = MergeRepoConfigs(user.Repos[repoName], repo)
			}
		}
	}
}
