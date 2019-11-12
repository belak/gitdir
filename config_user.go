package gitdir

import (
	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

func (c *Config) loadUserConfigs() error {
	// Clear out any existing user configs
	c.users = make(map[string]*models.UserConfig)

	// Bail early if we don't need to load anything.
	if !c.adminConfig.Options.UserConfigKeys && !c.adminConfig.Options.UserConfigRepos {
		return nil
	}

	var errors []error

	for username := range c.adminConfig.Users {
		errors = append(errors, c.loadUserInternal(username, ""))
	}

	// Because we want to display all the errors, we return this as a
	// multi-error rather than bailing on the first error.
	return newMultiError(errors...)
}

func (c *Config) LoadUser(username string, hash string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.loadUserInternal(username, hash)
}

func (c *Config) loadUserInternal(username string, hash string) error {
	userRepo, err := git.EnsureRepo(c.fs, "admin/user-"+username)
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

		c.users[username] = userConfig
	}

	c.flatten()

	return nil
}
