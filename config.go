package gitdir

import (
	"fmt"

	billy "github.com/go-git/go-billy/v5"

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

	// Internal state
	fs         billy.Filesystem
	publicKeys map[string]string `yaml:"-"`

	// We store any override hashes for repos so this can be used for hooks as
	// well.
	adminRepoHash string
	orgRepos      map[string]string
	userRepos     map[string]string
}

// NewConfig returns an empty config, attached to the given fs. In general, Load
// should be called after creating a new config at a bare minimum.
func NewConfig(fs billy.Filesystem) *Config {
	return &Config{
		Invites: make(map[string]string),
		Groups:  make(map[string][]string),
		Orgs:    make(map[string]*models.OrgConfig),
		Users:   make(map[string]*models.AdminConfigUser),
		Repos:   make(map[string]*models.RepoConfig),

		orgRepos:   make(map[string]string),
		userRepos:  make(map[string]string),
		publicKeys: make(map[string]string),

		Options: models.DefaultAdminConfigOptions,

		fs: fs,
	}
}

// Load will load the config from the given fs.
func (c *Config) Load() error {
	adminRepo, err := c.openAdminRepo()
	if err != nil {
		return err
	}

	return c.loadConfig(adminRepo)
}

func (c *Config) openAdminRepo() (*git.Repository, error) {
	adminRepo, err := git.EnsureRepo(c.fs, "admin/admin")
	if err != nil {
		return nil, err
	}

	err = adminRepo.Checkout(c.adminRepoHash)
	if err != nil {
		return nil, err
	}

	return adminRepo, nil
}

func (c *Config) loadConfig(adminRepo *git.Repository) error {
	// Load config
	err := c.loadAdminConfig(adminRepo)
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

	c.flatten()

	return nil
}

func (c *Config) EnsureConfig() error {
	adminRepo, err := c.openAdminRepo()
	if err != nil {
		return err
	}

	return c.ensureConfig(adminRepo)
}

func (c *Config) ensureConfig(adminRepo *git.Repository) error {
	// Ensure config
	err := c.ensureAdminConfig(adminRepo)
	if err != nil {
		return err
	}

	// Load config
	err = c.loadConfig(adminRepo)
	if err != nil {
		return err
	}

	// We only commit at the very end, after everything has been loaded. This
	// ensures we have a valid config.
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

	return nil
}

// EnsureUser will load the current admin config and ensure the given user
// exists.
func (c *Config) EnsureAdminUser(username string, pubKey *models.PublicKey) error {
	adminRepo, err := c.openAdminRepo()
	if err != nil {
		return err
	}

	// Ensure the base config before we try and add a user.
	err = c.ensureAdminConfig(adminRepo)
	if err != nil {
		return err
	}

	// Ensure User
	err = c.ensureAdminUser(adminRepo, username, pubKey)
	if err != nil {
		return err
	}

	// Load config
	err = c.loadConfig(adminRepo)
	if err != nil {
		return err
	}

	// We only commit at the very end, after everything has been loaded. This
	// ensures we have a valid config.
	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		err = adminRepo.Commit(fmt.Sprintf("Added %s to config as admin", username), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) flatten() {
	// Add all user public keys to the config.
	for username, user := range c.Users {
		for _, key := range user.Keys {
			c.publicKeys[key.RawMarshalAuthorizedKey()] = username
		}
	}
}

// SetHash will set the hash of the admin repo to use when loading.
func (c *Config) SetHash(hash string) error {
	adminRepo, err := git.EnsureRepo(c.fs, "admin/admin")
	if err != nil {
		return err
	}

	err = adminRepo.Checkout(hash)
	if err != nil {
		return err
	}

	c.adminRepoHash = hash

	return nil
}

// SetUserHash will set the hash of the given user repo to use when loading.
func (c *Config) SetUserHash(username, hash string) error {
	repo, err := git.EnsureRepo(c.fs, "admin/user-"+username)
	if err != nil {
		return err
	}

	err = repo.Checkout(hash)
	if err != nil {
		return err
	}

	c.userRepos[username] = hash

	return nil
}

// SetOrgHash will set the hash of the given org repo to use when loading.
func (c *Config) SetOrgHash(orgName, hash string) error {
	repo, err := git.EnsureRepo(c.fs, "admin/org-"+orgName)
	if err != nil {
		return err
	}

	err = repo.Checkout(hash)
	if err != nil {
		return err
	}

	c.orgRepos[orgName] = hash

	return nil
}
