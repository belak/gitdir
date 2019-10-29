package main

import (
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v3"
)

// AdminConfig is a combination of all the config types we're going to be
// loading. This is a root config meant to hold all the loaded configs in an
// easier to process format.
type AdminConfig struct {
	// These all come directly from the yaml files.
	Invites map[string]string
	Users   map[string]UserConfig
	Orgs    map[string]OrgConfig
	Repos   map[string]RepoConfig
	Groups  map[string][]string
	Options AdminOptionsConfig

	// Mapping of public key to username
	PublicKeys map[string][]string `yaml:"-"`
}

func newAdminConfig() *AdminConfig {
	return &AdminConfig{
		Invites: make(map[string]string),
		Users:   make(map[string]UserConfig),
		Orgs:    make(map[string]OrgConfig),
		Repos:   make(map[string]RepoConfig),
		Groups:  make(map[string][]string),

		PublicKeys: make(map[string][]string),

		// Defaults. These should be set in ensure config, but we have them here
		// for reference.
		Options: defaultAdminOptions,
	}
}

// AdminOptionsConfig contains all the server level settings which can be
// changed at runtime.
type AdminOptionsConfig struct {
	// GitUser refers to which username to use as the global git user.
	GitUser string `yaml:"git_user"`

	// OrgPrefix refers to the prefix to use when cloning org repos.
	OrgPrefix string `yaml:"org_prefix"`

	// UserPrefix refers to the prefix to use when cloning user repos.
	UserPrefix string `yaml:"user_prefix"`

	// InvitePrefix refers to the prefix to use when sshing in with an invite.
	InvitePrefix string `yaml:"invite_prefix"`

	// ImplicitRepos allows a user with admin access to that area to create
	// repos by simply pushing to them.
	ImplicitRepos bool `yaml:"implicit_repos"`

	// UserConfigKeys allows users to specify ssh keys in their own config,
	// rather than relying on the main admin config.
	UserConfigKeys bool `yaml:"user_config_keys"`

	// UserConfigRepos allows users to specify repos in their own config, rather
	// than relying on the main admin config.
	UserConfigRepos bool `yaml:"user_config_repos"`

	// OrgConfig allows org admins to configure orgs in their own config, rather
	// than relying on the main admin config.
	OrgConfig bool `yaml:"org_config"`

	// OrgConfigRepos allows org admins to specify repos in their own config,
	// rather than relying on the main admin config.
	OrgConfigRepos bool `yaml:"org_config_repos"`
}

var defaultAdminOptions = AdminOptionsConfig{
	GitUser:      "git",
	OrgPrefix:    "@",
	UserPrefix:   "~",
	InvitePrefix: "invite:",
}

// LoadAdminConfig loads an AdminConfig and PrivateKeys from the admin repo at
// admin/admin. An error should only be returned in the case of an
// unrecoverable error. If the server can start up at all, this should return
// nil and log any relevant warnings.
func LoadAdminConfig() (*AdminConfig, []PrivateKey, error) {
	ret := newAdminConfig()

	// Step 1: open the admin repo
	adminRepo, err := EnsureRepo("admin/admin", true)
	if err != nil {
		// Most config repos are "optional", but if the admin repo can't even be
		// created, we've got a big problem.
		return nil, nil, err
	}

	// Step 2: load settings from the admin repo - if any of these failed, we
	// can kill the server. In general, if they failed, it means something
	// happened at the git repo level or it's an invalid config.
	err = ret.loadAdminRepo(adminRepo)
	if err != nil {
		return nil, nil, err
	}

	pks, err := loadAdminSSHKeys(adminRepo)
	if err != nil {
		return nil, nil, err
	}

	// Step 3: Load the user and org configs from their respective config config
	// repos and merge them with the root config. Note that we ignore any errors
	// here because we only want admin errors to cause issues.
	ret.loadUserConfigs()
	ret.loadOrgConfigs()

	// Step 5: Validation

	// Step 6: Normalization
	err = ret.normalize()
	if err != nil {
		return nil, nil, err
	}

	// Step 7: Ensure all repos
	err = ret.ensureRepos()
	if err != nil {
		return nil, nil, err
	}

	return ret, pks, nil
}

func (ac *AdminConfig) loadAdminRepo(adminRepo *WorkingRepo) error {
	err := adminRepo.UpdateFile("config.yml", ensureSampleConfig)
	if err != nil {
		return err
	}

	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		err = adminRepo.Commit("Updated config.yml", nil)
		if err != nil {
			return err
		}
	}

	data, err := adminRepo.GetFile("config.yml")
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, ac)
}

func (ac *AdminConfig) normalize() error {
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

	for key := range ac.PublicKeys {
		ac.PublicKeys[key] = sliceUniqMap(ac.PublicKeys[key])
	}

	return nil
}

func (ac *AdminConfig) ensureRepos() error {
	var repos []string

	for repoName := range ac.Repos {
		repos = append(repos, "top-level/"+repoName)
	}

	for userName, user := range ac.Users {
		for repoName := range user.Repos {
			repos = append(repos, "users/"+userName+"/"+repoName)
		}
	}

	for orgName, org := range ac.Orgs {
		for repoName := range org.Repos {
			repos = append(repos, "orgs/"+orgName+"/"+repoName)
		}
	}

	for _, repo := range repos {
		_, err := EnsureRepo(repo, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadRSAKey(adminRepo *WorkingRepo) (PrivateKey, error) {
	rsaData, err := adminRepo.GetFile("keys/id_rsa")
	if err != nil {
		var pk PrivateKey

		log.Warn().Msg("Regenerating key: keys/id_rsa missing")

		pk, err = GenerateRSAKey()
		if err != nil {
			return nil, err
		}

		rsaData, err = pk.MarshalPrivateKey()
		if err != nil {
			return nil, err
		}

		err = adminRepo.CreateFile("keys/id_rsa", rsaData)
		if err != nil {
			return nil, err
		}
	}

	rsaKey, err := ParseRSAKey(rsaData)
	if err != nil {
		return nil, err
	}

	return rsaKey, err
}

func loadEd25519Key(adminRepo *WorkingRepo) (PrivateKey, error) {
	ed25519Data, err := adminRepo.GetFile("keys/id_ed25519")
	if err != nil {
		var pk PrivateKey

		log.Warn().Msg("Regenerating key: keys/id_ed25519 missing")

		pk, err = GenerateEd25519Key()
		if err != nil {
			return nil, err
		}

		ed25519Data, err = pk.MarshalPrivateKey()
		if err != nil {
			return nil, err
		}

		err = adminRepo.CreateFile("keys/id_ed25519", ed25519Data)
		if err != nil {
			return nil, err
		}
	}

	ed25519Key, err := ParseEd25519Key(ed25519Data)
	if err != nil {
		return nil, err
	}

	return ed25519Key, err
}

func loadAdminSSHKeys(adminRepo *WorkingRepo) ([]PrivateKey, error) {
	// Load the ssh keys from the admin repo. We want these to be available even
	// if there are config errors. However, even if this fails, it's not the end
	// of the world. The SSH libraries we use will auto-generate keys if they
	// don't exist at runtime.
	var pks []PrivateKey

	rsaKey, err := loadRSAKey(adminRepo)
	if err != nil {
		return nil, err
	}

	pks = append(pks, rsaKey)

	ed25519Key, err := loadEd25519Key(adminRepo)
	if err != nil {
		return nil, err
	}

	pks = append(pks, ed25519Key)

	// If the worktree isn't clean, the keys have been updated, so we need to
	// commit them.
	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return nil, err
	}

	if !status.IsClean() {
		err = adminRepo.Commit("Updated ssh keys", nil)
		if err != nil {
			return nil, err
		}
	}

	return pks, nil
}
