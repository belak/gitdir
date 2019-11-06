package models

// RepoConfig represents the values under repos in the main admin config, any
// org configs, or any user configs.
type RepoConfig struct {
	// Public allows any user of the service to access this repository for
	// reading
	Public bool

	// Any user or group who explicitly has write access
	Write []string

	// Any user or group who explicitly has read access
	Read []string
}

// NewRepoConfig returns a blank RepoConfig.
func NewRepoConfig() *RepoConfig {
	return &RepoConfig{}
}
