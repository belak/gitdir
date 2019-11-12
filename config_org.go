package gitdir

import (
	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

func (c *Config) loadOrgConfigs() error {
	// Clear out any existing org configs
	c.orgs = make(map[string]*models.OrgConfig)

	// Bail early if we don't need to load anything.
	if !c.adminConfig.Options.OrgConfig {
		return nil
	}

	var errors []error

	for orgName := range c.adminConfig.Orgs {
		errors = append(errors, c.loadOrgInternal(orgName, ""))
	}

	// Because we want to display all the errors, we return this as a
	// multi-error rather than bailing on the first error.
	return newMultiError(errors...)
}

func (c *Config) LoadOrg(orgName string, hash string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.loadOrgInternal(orgName, hash)
}

func (c *Config) loadOrgInternal(orgName string, hash string) error {
	orgRepo, err := git.EnsureRepo(c.fs, "admin/org-"+orgName)
	if err != nil {
		return err
	}

	err = orgRepo.Checkout(hash)
	if err != nil {
		return err
	}

	data, err := orgRepo.GetFile("config.yml")
	if err != nil {
		return err
	}

	orgConfig, err := models.ParseOrgConfig(data)
	if err != nil {
		return err
	}

	c.orgs[orgName] = orgConfig

	c.flatten()

	return nil
}
