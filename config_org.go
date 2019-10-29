package main

import (
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v3"
)

type OrgConfig struct {
	// TODO: org groups

	Admin []string
	Write []string
	Read  []string
	Repos map[string]RepoConfig
}

func loadOrg(orgName string) (OrgConfig, error) {
	oc := OrgConfig{
		Repos: make(map[string]RepoConfig),
	}

	orgRepo, err := EnsureRepo("admin/org-"+orgName, true)
	if err != nil {
		return oc, err
	}

	data, err := orgRepo.GetFile("config.yml")
	if err != nil {
		return oc, err
	}

	err = yaml.Unmarshal(data, &oc)

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
