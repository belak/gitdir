package main

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

// MergeRepoConfigs flattens a number of repo configs for the same repo into
// one RepoConfig.
func MergeRepoConfigs(rcList ...RepoConfig) RepoConfig {
	var root RepoConfig

	for _, rc := range rcList {
		root.Write = append(root.Write, rc.Write...)
		root.Read = append(root.Read, rc.Read...)
	}

	root.Write = sliceUniqMap(root.Write)
	root.Read = sliceUniqMap(root.Read)

	return root
}
