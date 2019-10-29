package main

import (
	"strings"

	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v3"
)

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
	Options AdminOptionsConfig

	// Mapping of public key to username
	PublicKeys map[string][]string `yaml:"-"`
}

type AdminOptionsConfig struct {
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

func LoadAdminConfig() (*AdminConfig, []PrivateKey, error) {
	ret := &AdminConfig{
		Users:  make(map[string]UserConfig),
		Orgs:   make(map[string]OrgConfig),
		Repos:  make(map[string]RepoConfig),
		Groups: make(map[string][]string),

		PublicKeys: make(map[string][]string),
	}

	// Step 1: open the admin repo
	adminRepo, err := EnsureRepo("admin/admin", true)
	if err != nil {
		log.Error().Err(err).Str("repo_path", "admin/admin").Msg("Failed to open admin repo")

		// Most config repos are "optional", but if the admin repo can't even be
		// created, we've got a big problem.
		return nil, nil, err
	}

	// Step 2: load settings from the admin repo - if any of these failed, we
	// can kill the server.
	err = ret.loadAdminRepo(adminRepo)
	if err != nil {
		return nil, nil, err
	}

	pks, err := loadAdminSSHKeys(adminRepo)
	if err != nil {
		return nil, nil, err
	}

	err = ret.loadAdminUserKeys(adminRepo)
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
	err = ret.Normalize()
	if err != nil {
		return nil, nil, err
	}

	// Step 7: Ensure all repos
	err = ret.EnsureRepos()
	if err != nil {
		return nil, nil, err
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

	for key := range ac.PublicKeys {
		ac.PublicKeys[key] = sliceUniqMap(ac.PublicKeys[key])
	}

	return nil
}

func (ac *AdminConfig) EnsureRepos() error {
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

func (ac *AdminConfig) loadAdminRepo(adminRepo *WorkingRepo) error {
	data, err := adminRepo.GetFile("config.yml")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load settings")

		// Set data to our sample config
		data = []byte(sampleConfig)

		// If we failed to load config, we can update the config with our
		// own sample config.
		err = adminRepo.CreateFile("config.yml", data)
		if err != nil {
			return err
		}

		err = adminRepo.Commit("Added sample config.yml", nil)
		if err != nil {
			return err
		}
	}

	return yaml.Unmarshal(data, &ac)
}

func (ac *AdminConfig) loadAdminUserKeys(adminRepo *WorkingRepo) error {
	usersDir, err := adminRepo.WorktreeFS.Chroot("users")
	if err != nil {
		return err
	}

	entries, err := usersDir.ReadDir(".")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// If it's a directory, we want to load all the keys under this dir. If
		// it's a file (and it's a valid name), load this specific file.
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// If it ends in .pub, chop off the pub from the name.
		var username string
		if strings.HasSuffix(filename, ".pub") {
			username = filename[:len(filename)-4]
		} else {
			username = filename
		}

		// Make sure this user is defined in the admin config.
		if _, ok := ac.Users[username]; !ok {
			ac.Users[username] = UserConfig{
				Repos: make(map[string]RepoConfig),
			}
		}

		f, err := usersDir.Open(filename)
		if err != nil {
			return err
		}

		pks, err := loadAuthorizedKeys(f)
		if err != nil {
			return err
		}

		for _, key := range pks {
			mkey := key.RawMarshalAuthorizedKey()
			ac.PublicKeys[mkey] = append(ac.PublicKeys[mkey], username)
		}
	}

	return nil
}
