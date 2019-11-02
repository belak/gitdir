package gitdir

import (
	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

func (c *Config) loadOrgConfigs() error {
	// Bail early if we don't need to load anything.
	if !c.Options.OrgConfig {
		return nil
	}

	var errors []error

	for orgName := range c.Orgs {
		errors = append(errors, c.loadOrgConfig(orgName))
	}

	// Because we want to display all the errors, we return this as a
	// multi-error rather than bailing on the first error.
	return newMultiError(errors...)
}

func (c *Config) loadOrgConfig(orgName string) error {
	orgRepo, err := git.EnsureRepo(c.fs, "admin/org-"+orgName)
	if err != nil {
		return err
	}

	err = orgRepo.Checkout(c.orgRepos[orgName])
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

	c.Orgs[orgName].Admin = append(c.Orgs[orgName].Admin, orgConfig.Admin...)
	c.Orgs[orgName].Write = append(c.Orgs[orgName].Write, orgConfig.Write...)
	c.Orgs[orgName].Read = append(c.Orgs[orgName].Read, orgConfig.Read...)

	if c.Options.OrgConfigRepos {
		for repoName, repo := range orgConfig.Repos {
			// If it's already defined, skip it.
			//
			// TODO: this should throw a validation error
			if _, ok := c.Orgs[orgName].Repos[repoName]; ok {
				continue
			}

			c.Orgs[orgName].Repos[repoName] = repo
		}
	}

	return nil
}
