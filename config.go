package gitdir

import (
	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

// Config represents the config which has been loaded from all repos.
type Config struct {
	Invites     map[string]string
	Groups      map[string][]string
	Orgs        map[string]*models.OrgConfig
	Users       map[string]*models.AdminConfigUser
	Repos       map[string]*models.RepoConfig
	Options     models.AdminConfigOptions
	PrivateKeys []models.PrivateKey

	publicKeys map[string]string `yaml:"-"`
}

func newConfig() *Config {
	return &Config{
		Invites: make(map[string]string),
		Groups:  make(map[string][]string),
		Orgs:    make(map[string]*models.OrgConfig),
		Users:   make(map[string]*models.AdminConfigUser),
		Repos:   make(map[string]*models.RepoConfig),
		Options: models.DefaultAdminConfigOptions,

		publicKeys: make(map[string]string),
	}
}

func LoadConfig(
	baseDir string,
	adminHash git.Hash,
	orgHashes map[string]git.Hash,
	userHashes map[string]git.Hash,
) (*Config, error) {
	c := newConfig()

	if orgHashes == nil {
		orgHashes = make(map[string]git.Hash)
	}

	if userHashes == nil {
		userHashes = make(map[string]git.Hash)
	}

	adminRepo, err := git.EnsureRepo(baseDir, "admin/admin")
	if err != nil {
		return nil, err
	}

	err = adminRepo.Checkout(adminHash)
	if err != nil {
		return nil, err
	}

	// Ensure config
	//
	// TODO: this should not be here because it is also used in hooks.
	err = c.ensureAdminConfig(adminRepo)
	if err != nil {
		return nil, err
	}

	// Load config
	err = c.loadAdminConfig(adminRepo)
	if err != nil {
		return nil, err
	}

	// Load sub-configs
	err = newMultiError(
		c.loadUserConfigs(baseDir, userHashes),
		c.loadOrgConfigs(baseDir, orgHashes),
	)
	if err != nil {
		return nil, err
	}

	// We actually only commit at the very end, after everything has been
	// loaded. This ensures we have a valid config.
	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return nil, err
	}

	if !status.IsClean() {
		err = adminRepo.Commit("Updated config", nil)
		if err != nil {
			return nil, err
		}
	}

	c.flatten()

	return c, nil
}

func (c *Config) flatten() {
	// Add all user public keys to the config.
	for username, user := range c.Users {
		for _, key := range user.Keys {
			c.publicKeys[key.RawMarshalAuthorizedKey()] = username
		}
	}
}
