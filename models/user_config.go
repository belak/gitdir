package models

import (
	"gopkg.in/yaml.v3"
)

// UserConfig represents the values under users in the main admin config or the
// contents of the config file in the user config repo. This type contains
// values shared between the different config types.
type UserConfig struct {
	Repos map[string]*RepoConfig `yaml:"repos"`
	Keys  []PublicKey            `yaml:"keys"`
}

// NewUserConfig returns a new, empty UserConfig.
func NewUserConfig() *UserConfig {
	return &UserConfig{
		Repos: make(map[string]*RepoConfig),
	}
}

// ParseUserConfig will return an UserConfig parsed from the given data. No
// additional validation is done.
func ParseUserConfig(data []byte) (*UserConfig, error) {
	uc := NewUserConfig()

	err := yaml.Unmarshal(data, uc)
	if err != nil {
		return nil, err
	}

	return uc, nil
}
