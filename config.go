package gitdir

import (
	"sync"

	"gopkg.in/src-d/go-billy.v4"

	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

// Config represents the config which has been loaded from all repos.
type Config struct {
	fs   billy.Filesystem
	lock *sync.RWMutex

	// Internal state
	adminConfig *models.AdminConfig
	orgs        map[string]*models.OrgConfig
	users       map[string]*models.UserConfig
	pks         []models.PrivateKey

	// Cache
	publicKeys map[string]string `yaml:"-"`
}

// NewConfig returns an empty config, attached to the given fs. In general, this
// is designed to be called in order:
//
// c := NewConfig(fs)
// err := c.Load()
// if err != nil {
//	return err
// }
// err = c.Validate()
// if err != nil {
//	return err
// }
func NewConfig(fs billy.Filesystem) *Config {
	return &Config{
		fs:   fs,
		lock: &sync.RWMutex{},

		// Start with a blank admin config.
		adminConfig: models.NewAdminConfig(),

		// All loaded user configs
		orgs:  make(map[string]*models.OrgConfig),
		users: make(map[string]*models.UserConfig),

		// Cache of all public keys
		publicKeys: make(map[string]string),
	}
}

// Load will load the config at the given hash.
func (c *Config) Load(hash string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	adminRepo, err := git.EnsureRepo(c.fs, "admin/admin")
	if err != nil {
		return err
	}

	err = adminRepo.Checkout(hash)
	if err != nil {
		return err
	}

	// Ensure config
	//
	// TODO: this should not be here because it is also used in hooks.
	err = c.ensureAdminConfig(adminRepo)
	if err != nil {
		return err
	}

	// Load config
	err = c.loadAdminConfig(adminRepo)
	if err != nil {
		return err
	}

	// Load sub-configs
	err = newMultiError(
		c.loadUserConfigs(),
		c.loadOrgConfigs(),
	)
	if err != nil {
		return err
	}

	// We actually only commit at the very end, after everything has been
	// loaded. This ensures we have a valid config.
	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		err = adminRepo.Commit("Updated config", nil)
		if err != nil {
			return err
		}
	}

	c.flatten()

	return nil
}

func (c *Config) flatten() {
	// Clear out all existing public keys.
	c.publicKeys = make(map[string]string)

	// Add all user public keys to the config.
	if c.adminConfig.Options.UserConfigKeys {
		for username, user := range c.adminConfig.Users {
			for _, key := range user.Keys {
				c.publicKeys[key.RawMarshalAuthorizedKey()] = username
			}
		}
	}

	for username, user := range c.adminConfig.Users {
		for _, key := range user.Keys {
			// TODO: warn if there are any conflicting keys
			c.publicKeys[key.RawMarshalAuthorizedKey()] = username
		}
	}
}

func (c *Config) lookupOrgConfig(orgName string) (*models.OrgConfig, bool) {
	ret, ok := c.adminConfig.Orgs[orgName]
	if !ok {
		return models.NewOrgConfig(), false
	}

	if c.adminConfig.Options.OrgConfig {
		orgConfig, ok := c.orgs[orgName]
		if !ok {
			return ret, false
		}

		ret.Admin = append(ret.Admin, orgConfig.Admin...)
		ret.Write = append(ret.Write, orgConfig.Write...)
		ret.Read = append(ret.Read, orgConfig.Read...)
		ret.Repos = models.MergeRepoMaps(ret.Repos, orgConfig.Repos)
	}

	return ret, true
}

func (c *Config) lookupUserConfig(username string) (*models.UserConfig, bool) {
	ret := models.NewUserConfig()

	adminUser, ok := c.adminConfig.Users[username]
	if !ok {
		return ret, false
	}

	// Initial values can just be a reference. Because the types don't line up,
	// we unfortunately can't just replace the value.
	ret.Repos = adminUser.Repos
	ret.Keys = adminUser.Keys

	userConfig, ok := c.users[username]
	if !ok {
		// We have a valid config at this point if we don't need to load
		// anything from the user config.
		return ret, !(c.adminConfig.Options.UserConfigRepos || c.adminConfig.Options.UserConfigKeys)
	}

	if c.adminConfig.Options.UserConfigRepos {
		ret.Repos = models.MergeRepoMaps(ret.Repos, userConfig.Repos)
	}

	if c.adminConfig.Options.UserConfigKeys {
		ret.Keys = append(ret.Keys, userConfig.Keys...)
	}

	return ret, true
}
