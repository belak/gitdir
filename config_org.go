package main

import (
	"github.com/rs/zerolog/log"
)

// OrgConfig represents the values under orgs in the main admin config or the
// contents of the config file in the org config repo.
type OrgConfig struct {
	Admin []string
	Write []string
	Read  []string
	Repos map[string]RepoConfig
}

func loadOrg(orgName string) (OrgConfig, error) {
	oc := OrgConfig{
		Repos: make(map[string]RepoConfig),
	}

	err := loadConfig("admin/org-"+orgName, &oc)

	return oc, err
}

func (ac *AdminConfig) loadOrgConfigs() {
	if !ac.Options.OrgConfig {
		return
	}

	tmpOrgs := make(map[string]OrgConfig)

	for orgName, org := range ac.Orgs {
		if org.Repos == nil {
			org.Repos = make(map[string]RepoConfig)
		}

		orgConfig, err := loadOrg(orgName)
		if err != nil {
			log.Warn().Err(err).Str("org", orgName).Msg("Failed to load org repo")
			continue
		}

		// If they can't load repos from org configs, we need to ignore them
		if !ac.Options.OrgConfigRepos {
			orgConfig.Repos = make(map[string]RepoConfig)
		}

		tmpOrgs[orgName] = orgConfig
	}

	for orgName, org := range tmpOrgs {
		ac.Orgs[orgName] = MergeOrgConfigs(ac.Orgs[orgName], org)
	}
}

// MergeOrgConfigs flattens a number of org configs for the same org into one
// OrgConfig.
func MergeOrgConfigs(orgList ...OrgConfig) OrgConfig {
	root := OrgConfig{
		Repos: make(map[string]RepoConfig),
	}

	for _, oc := range orgList {
		root.Admin = append(root.Admin, oc.Admin...)
		root.Write = append(root.Write, oc.Write...)
		root.Read = append(root.Read, oc.Read...)

		for repoName, repo := range oc.Repos {
			root.Repos[repoName] = MergeRepoConfigs(root.Repos[repoName], repo)
		}
	}

	root.Admin = sliceUniqMap(root.Admin)
	root.Write = sliceUniqMap(root.Write)
	root.Read = sliceUniqMap(root.Read)

	return root
}
