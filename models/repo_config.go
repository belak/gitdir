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

func MergeRepoConfigs(configs ...*RepoConfig) *RepoConfig {
	var found bool

	ret := NewRepoConfig()

	for _, config := range configs {
		if config == nil {
			continue
		}

		found = true

		ret.Public = ret.Public || config.Public
		ret.Write = append(ret.Write, config.Write...)
		ret.Read = append(ret.Read, config.Read...)
	}

	if !found {
		return nil
	}

	return ret
}

func MergeRepoMaps(configs ...map[string]*RepoConfig) map[string]*RepoConfig {
	ret := make(map[string]*RepoConfig)

	for _, config := range configs {
		for repoName, repo := range config {
			retRepo := MergeRepoConfigs(ret[repoName], repo)
			if retRepo == nil {
				continue
			}

			ret[repoName] = retRepo
		}
	}

	return ret
}
