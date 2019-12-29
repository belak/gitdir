package gitdir

import (
	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

func (c *Config) loadUserConfigs(baseDir string, hashes map[string]git.Hash) error {
	// Bail early if we don't need to load anything.
	if !c.Options.UserConfigKeys && !c.Options.UserConfigRepos {
		return nil
	}

	var errors []error

	for username := range c.Users {
		hash := hashes[username]
		errors = append(errors, c.loadUserConfig(baseDir, username, hash))
	}

	// Because we want to display all the errors, we return this as a
	// multi-error rather than bailing on the first error.
	return newMultiError(errors...)
}

func (c *Config) loadUserConfig(baseDir string, username string, hash git.Hash) error {
	userRepo, err := git.EnsureRepo(baseDir, "admin/user-"+username)
	if err != nil {
		return err
	}

	err = userRepo.Checkout(hash)
	if err != nil {
		return err
	}

	if userRepo.FileExists("config.yml") {
		data, err := userRepo.GetFile("config.yml")
		if err != nil {
			return err
		}

		userConfig, err := models.ParseUserConfig(data)
		if err != nil {
			return err
		}

		if c.Options.UserConfigKeys {
			c.Users[username].Keys = append(c.Users[username].Keys, userConfig.Keys...)
		}

		if c.Options.UserConfigRepos {
			for repoName, repo := range userConfig.Repos {
				// If it's already defined, skip it.
				//
				// TODO: this should throw a validation error
				if _, ok := c.Users[username].Repos[repoName]; ok {
					continue
				}

				c.Users[username].Repos[repoName] = repo
			}
		}
	}

	return nil
}
