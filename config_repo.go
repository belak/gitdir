package main

type RepoConfig struct {
	// Public allows any user of the service to access this repository for
	// reading
	Public bool

	// Any user or group who explicitly has write access
	Write []string

	// Any user or group who explicitly has read access
	Read []string
}

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
