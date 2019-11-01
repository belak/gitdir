package models

import (
	"gopkg.in/yaml.v3"
)

// OrgConfig represents the values under orgs in the main admin config or the
// contents of the config file in the org config repo.
type OrgConfig struct {
	Admin []string               `yaml:"admin"`
	Write []string               `yaml:"write"`
	Read  []string               `yaml:"read"`
	Repos map[string]*RepoConfig `yaml:"repos"`
}

// NewOrgConfig returns a new, empty OrgConfig
func NewOrgConfig() *OrgConfig {
	return &OrgConfig{
		Repos: make(map[string]*RepoConfig),
	}
}

// ParseOrgConfig will return an OrgConfig parsed from the given data. No
// additional validation is done.
func ParseOrgConfig(data []byte) (*OrgConfig, error) {
	oc := NewOrgConfig()

	err := yaml.Unmarshal(data, oc)
	if err != nil {
		return nil, err
	}

	return oc, nil
}
