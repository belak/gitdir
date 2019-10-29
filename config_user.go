package main

import (
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v3"
)

type UserConfig struct {
	// TODO: user groups?

	IsAdmin  bool                  `yaml:"is_admin"`
	Disabled bool                  `yaml:"disabled"`
	Repos    map[string]RepoConfig `yaml:"repos"`
	Keys     []PublicKey           `yaml:"keys"`
}

func loadUser(username string) (UserConfig, []PublicKey, error) {
	uc := UserConfig{
		Repos: make(map[string]RepoConfig),
	}

	userRepo, err := EnsureRepo("admin/user-"+username, true)
	if err != nil {
		return uc, nil, err
	}

	data, err := userRepo.GetFile("config.yml")
	if err != nil {
		return uc, nil, err
	}

	err = yaml.Unmarshal(data, &uc)
	if err != nil {
		return uc, nil, err
	}

	f, err := userRepo.WorktreeFS.Open("authorized_keys")
	if err != nil {
		return uc, nil, err
	}

	pks, err := loadAuthorizedKeys(f)
	if err != nil {
		return uc, nil, err
	}

	return uc, pks, nil
}

func (ac *AdminConfig) loadUserConfigs() {
	// If we have no reason to load config from user repos, we can bail early.
	if !ac.Options.UserConfigRepos && !ac.Options.UserConfigKeys {
		return
	}

	for username, user := range ac.Users {
		if user.Repos == nil {
			user.Repos = make(map[string]RepoConfig)
		}

		userConfig, userKeys, err := loadUser(username)

		if ac.Options.UserConfigKeys {
			// Add all the user keys - we actually do this before handling the error
			// so if the user breaks their config, they can still hopefully SSH in
			// to fix it.
			for _, key := range userKeys {
				mkey := string(key.Marshal())
				ac.PublicKeys[mkey] = append(ac.PublicKeys[mkey], username)
			}
		}

		if err != nil {
			log.Warn().Err(err).Str("username", username).Msg("Failed to load user repo")
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
