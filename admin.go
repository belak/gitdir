package main

import (
	"errors"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v3"
)

type User struct {
	Username    string
	IsAnonymous bool
	IsAdmin     bool
}

func (ac *AdminConfig) GetUserFromKey(pk PublicKey) (*User, error) {
	username, ok := ac.PublicKeys[string(pk.Marshal())]
	if !ok {
		return AnonymousUser, errors.New("key does not match a user")
	}

	userConfig, ok := ac.Users[username]
	if !ok {
		return AnonymousUser, errors.New("key does not match a user")
	}

	return &User{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}

var AnonymousUser = &User{
	Username:    "<anonymous>",
	IsAnonymous: true,
}

type RepoConfig struct {
	Write []string
	Read  []string
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

type UserConfig struct {
	// TODO: user groups?

	IsAdmin  bool                  `yaml:"is_admin"`
	Disabled bool                  `yaml:"disabled"`
	Repos    map[string]RepoConfig `yaml:"repos"`
}

type OrgConfig struct {
	// TODO: org groups

	Admin []string
	Write []string
	Read  []string
	Repos map[string]RepoConfig
}

type OptionsConfig struct {
	// ImplicitRepos allows a user with write access to create repos by simply
	// cloning or pushing to them.
	ImplicitRepos bool `yaml:"implicit_repos"`

	// UserConfigRepos allows users to specify repos in their own config, rather
	// than relying on the main admin config.
	UserConfigRepos bool `yaml:"user_config_repos"`

	// OrgConfig allows org admins to configure orgs in their own config, rather
	// than relying on the main admin config.
	OrgConfig bool `yaml:"org_config"`

	// OrgConfigRepos allows org admins to specify repos in their own config,
	// rather than relying on the main admin config.
	OrgConfigRepos bool `yaml:"org_config_repos"`

	// OrgConfigUsers allows org admins to specify users in their own config,
	// rather than relying on the main admin config.
	OrgConfigUsers bool `yaml:"org_config_users"` // TODO: implement this
}

// This is a combination of all the config types we're going to be loading. This
// is a root config meant to hold all the loaded configs in an easier to process
// format.
type AdminConfig struct {
	// TODO: global read/write? is this only for top level repos?

	// These all come directly from the yaml files.
	Users   map[string]UserConfig
	Orgs    map[string]OrgConfig
	Repos   map[string]RepoConfig
	Groups  map[string][]string
	Options OptionsConfig

	// Mapping of public key to username
	PublicKeys map[string]string `yaml:"-"`
}

func (c *Config) LoadSettings() (*AdminConfig, []PrivateKey, error) {
	ret := &AdminConfig{
		Users:  make(map[string]UserConfig),
		Orgs:   make(map[string]OrgConfig),
		Repos:  make(map[string]RepoConfig),
		Groups: make(map[string][]string),

		PublicKeys: make(map[string]string),
	}

	// Step 1: load the admin repo
	adminRepo, err := c.EnsureRepo("admin/admin", true)
	if err != nil {
		log.Error().Err(err).Str("repo_path", "admin/admin").Msg("Failed to open admin repo")

		// Most config repos are "optional", but if the admin repo can't even be
		// created, we've got a big problem.
		return nil, nil, err
	}

	// Step 2: Load the ssh keys from the admin repo. We want these to be
	// available even if there are config errors. However, even if this fails,
	// it's not the end of the world. The SSH libraries we use will
	// auto-generate keys if they don't exist at runtime.
	var pks []PrivateKey
	rsaData, err := adminRepo.GetFile("keys/id_rsa")
	if err != nil {
		log.Warn().Msg("Regenerating key: keys/id_rsa missing")

		pk, err := GenerateRSAKey()
		if err != nil {
			return nil, nil, err
		}
		rsaData, err = pk.MarshalPrivateKey()
		if err != nil {
			return nil, nil, err
		}
		err = adminRepo.CreateFile("keys/id_rsa", rsaData)
		if err != nil {
			return nil, nil, err
		}
	}
	rsaKey, err := ParseRSAKey(rsaData)
	if err != nil {
		return nil, nil, err
	}
	pks = append(pks, rsaKey)

	ed25519Data, err := adminRepo.GetFile("keys/id_ed25519")
	if err != nil {
		log.Warn().Msg("Regenerating key: keys/id_ed25519 missing")

		pk, err := GenerateEd25519Key()
		if err != nil {
			return nil, nil, err
		}
		ed25519Data, err = pk.MarshalPrivateKey()
		if err != nil {
			return nil, nil, err
		}
		err = adminRepo.CreateFile("keys/id_ed25519", ed25519Data)
		if err != nil {
			return nil, nil, err
		}
	}
	ed25519Key, err := ParseEd25519Key(ed25519Data)
	if err != nil {
		return nil, nil, err
	}
	pks = append(pks, ed25519Key)

	// If the worktree isn't clean, the keys have been updated, so we need to
	// commit them.
	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return nil, nil, err
	}
	if !status.IsClean() {
		err = adminRepo.Commit("Updated ssh keys", nil)
		if err != nil {
			return nil, nil, err
		}
	}

	// Step 3: load the root config from the admin repo
	data, err := adminRepo.GetFile("config.yml")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load settings")

		// Set data to our sample config
		data = []byte(sampleConfig)

		// If we failed to load config, we can update the config with our
		// own sample config.
		err = adminRepo.CreateFile("config.yml", data)
		if err != nil {
			return nil, nil, err
		}

		err = adminRepo.Commit("Added sample config.yml", nil)
		if err != nil {
			return nil, nil, err
		}
	}

	err = yaml.Unmarshal(data, &ret)
	if err != nil {
		return nil, nil, err
	}

	// Step 4: Load the user configs from user config repos and merge them with
	// the root config.
	for username, user := range ret.Users {
		if user.Repos == nil {
			user.Repos = make(map[string]RepoConfig)
		}

		userConfig, userKeys, err := c.loadUser(username)

		// Add all the user keys - we actually do this before handling the error
		// so if the user breaks their config, they can still SSH in to fix it.
		for _, key := range userKeys {
			// TODO: check for conflicts
			ret.PublicKeys[string(key.Marshal())] = username
		}

		if err != nil {
			log.Warn().Err(err).Str("username", username).Msg("Failed to load user repo")
			continue
		}

		if ret.Options.UserConfigRepos {
			// We only really need to merge repos when dealing with loading users,
			// as we don't want them to be able to set config options.
			for repoName, repo := range userConfig.Repos {
				user.Repos[repoName] = MergeRepoConfigs(user.Repos[repoName], repo)
			}
		}
	}

	// Step 5: Load the org configs from org config repos and merge them with
	// the root config.
	if ret.Options.OrgConfig {
		tmpOrgs := make(map[string]OrgConfig)
		for orgName, org := range ret.Orgs {
			if org.Repos == nil {
				org.Repos = make(map[string]RepoConfig)
			}

			orgConfig, err := c.loadOrg(orgName)
			if err != nil {
				log.Warn().Err(err).Str("org", orgName).Msg("Failed to load org repo")
				continue
			}

			// If they can't load repos from org configs, we need to ignore them
			if !ret.Options.OrgConfigRepos {
				orgConfig.Repos = make(map[string]RepoConfig)
			}

			tmpOrgs[orgName] = orgConfig
		}
		for orgName, org := range tmpOrgs {
			ret.Orgs[orgName] = MergeOrgConfigs(ret.Orgs[orgName], org)
		}
	}

	// Step 6: Validation

	// Step 7: Normalization
	err = ret.Normalize()
	if err != nil {
		return nil, nil, err
	}

	// Step 8: Ensure all repos
	var repos []string

	for repoName := range ret.Repos {
		repos = append(repos, "top-level/"+repoName)
	}

	for userName, user := range ret.Users {
		for repoName := range user.Repos {
			repos = append(repos, "users/"+userName+"/"+repoName)
		}
	}

	for orgName, org := range ret.Orgs {
		for repoName := range org.Repos {
			repos = append(repos, "orgs/"+orgName+"/"+repoName)
		}
	}

	for _, repo := range repos {
		_, err = c.EnsureRepo(repo, false)
		if err != nil {
			return nil, nil, err
		}
	}

	return ret, pks, nil
}

func (ac *AdminConfig) Normalize() error {
	for name := range ac.Groups {
		// Replace each of the groups with their expanded versions. This means
		// any future accesses won't need to recurse and so we can ignore the
		// error.
		expanded, err := groupMembers(ac.Groups, name, nil)
		if err != nil {
			return err
		}
		ac.Groups[name] = expanded
	}

	for repoKey, oldRepo := range ac.Repos {
		newRepo := oldRepo
		newRepo.Write = expandGroups(ac.Groups, newRepo.Write)
		newRepo.Read = expandGroups(ac.Groups, newRepo.Read)
		ac.Repos[repoKey] = newRepo
	}

	for orgKey, oldOrg := range ac.Orgs {
		newOrg := oldOrg
		newOrg.Admin = expandGroups(ac.Groups, newOrg.Admin)
		newOrg.Write = expandGroups(ac.Groups, newOrg.Write)
		newOrg.Read = expandGroups(ac.Groups, newOrg.Read)

		for repoKey, oldRepo := range newOrg.Repos {
			newRepo := oldRepo
			newRepo.Write = expandGroups(ac.Groups, newRepo.Write)
			newRepo.Read = expandGroups(ac.Groups, newRepo.Read)
			newOrg.Repos[repoKey] = newRepo
		}

		ac.Orgs[orgKey] = newOrg
	}

	return nil
}

func (c *Config) loadUser(username string) (UserConfig, []PublicKey, error) {
	uc := UserConfig{
		Repos: make(map[string]RepoConfig),
	}

	userRepo, err := c.EnsureRepo("admin/user-"+username, true)
	if err != nil {
		return uc, nil, err
	}

	data, err := userRepo.GetFile("config.yml")
	if err != nil {
		return uc, nil, err
	}

	err = yaml.Unmarshal(data, &uc)
	if err != nil {
		return uc, nil, err
	}

	keysDir, err := userRepo.WorktreeFS.ReadDir("keys")
	if err != nil {
		return uc, nil, err
	}

	var pks []PublicKey
	for _, entry := range keysDir {
		// We create .keep files in all the dirs to make it easier to ensure the
		// dirs exist.
		if entry.Name() == ".keep" {
			continue
		}

		data, err := userRepo.GetFile("keys/" + entry.Name())
		if err != nil {
			return uc, nil, err
		}

		pk, _, _, _, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			return uc, nil, err
		}

		pks = append(pks, PublicKey{pk, ""})
	}

	return uc, pks, nil
}

func (c *Config) loadOrg(orgName string) (OrgConfig, error) {
	oc := OrgConfig{
		Repos: make(map[string]RepoConfig),
	}

	orgRepo, err := c.EnsureRepo("admin/org-"+orgName, true)
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
