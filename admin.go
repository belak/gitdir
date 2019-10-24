package main

import (
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"strings"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-billy.v4"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	yaml "gopkg.in/yaml.v3"
)

type accessType int

type User struct {
	PublicKeys []publicKey `yaml:"keys"`
	IsAdmin    bool        `yaml:"is_admin"`

	// Data we don't want to load in from yaml
	Username    string `yaml:"-"`
	IsAnonymous bool   `yaml:"-"`
}

var AnonymousUser = &User{
	Username:    "<anonymous>",
	IsAnonymous: true,
}

func (a *AdminRepo) LoadUser(username string) (*User, error) {
	data, err := readFile(a.worktreeFS, "users/"+username+".yml")
	if err != nil {
		return nil, err
	}

	u := &User{Username: sanitize(username)}
	err = yaml.Unmarshal(data, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

type AdminRepo struct {
	*sync.RWMutex

	// We keep the worktree separate from the repo so we can still have a bare
	// repo. This also lets us do fun things like keep the worktree in memory.
	repo       *git.Repository
	worktree   *git.Worktree
	worktreeFS billy.Filesystem

	settings *adminConfig
	keys     keyConfig

	users    map[string]*User
	userKeys map[string]string
}

type keyConfig struct {
	Ed25519 ed25519.PrivateKey
	RSA     *rsa.PrivateKey
}

type repoConfig struct {
	// Permission levels
	Write []string
	Read  []string
}

type orgConfig struct {
	// Permission levels
	Admin []string
	Write []string
	Read  []string

	Repos map[string]repoConfig
}

type adminOptions struct {
	UserRepos bool `yaml:"user_repos"`
	UserKeys  bool `yaml:"user_keys"`

	OrgConfig            bool `yaml:"org_config"`
	OrgConfigPermissions bool `yaml:"org_config_permissions"`
	OrgConfigRepos       bool `yaml:"org_config_repos"`
	OrgConfigUsers       bool `yaml:"org_config_users"`
}

type adminConfig struct {
	Groups map[string][]string
	Repos  map[string]repoConfig
	Orgs   map[string]orgConfig

	Options adminOptions
}

func (c *Config) OpenAdminRepo() (*AdminRepo, error) {
	var err error

	a := &AdminRepo{
		RWMutex: &sync.RWMutex{},
	}

	a.repo, a.worktreeFS, err = c.EnsureRepo("admin/admin")
	if err != nil {
		return nil, err
	}

	err = a.Reload()
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Reload will clear the internal cache and re-load the keys and settings from a
// file. Note that this can be slow. If an error is returned, the original keys
// and settings will be kept.
func (a *AdminRepo) Reload() error {
	a.Lock()
	defer a.Unlock()

	var err error

	a.worktree, err = a.repo.Worktree()
	if err != nil {
		return err
	}

	// Reset the worktree to the head
	err = a.worktree.Checkout(&git.CheckoutOptions{
		Force: true,
	})
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return err
	}

	// Store the original values in case something fails to load
	kc := a.keys
	settings := a.settings
	users := a.users
	userKeys := a.userKeys

	// Reset the values and re-load them.
	a.keys = keyConfig{}
	a.settings = nil
	a.users = nil
	a.userKeys = nil

	// Load everything from the config again
	_, err = a.ensureSettings()
	if err != nil {
		a.keys = kc
		a.settings = settings
		a.users = users
		a.userKeys = userKeys
		return err
	}

	_, err = a.ensureServerKeys()
	if err != nil {
		a.keys = kc
		a.settings = settings
		a.users = users
		a.userKeys = userKeys
		return err
	}

	_, _, err = a.ensureUsers()
	if err != nil {
		a.keys = kc
		a.settings = settings
		a.users = users
		a.userKeys = userKeys
		return err
	}

	return nil
}

func (a *AdminRepo) GetSettings() (*adminConfig, error) {
	a.RLock()
	defer a.RUnlock()

	if a.settings == nil {
		return nil, errors.New("Settings not loaded")
	}

	return a.settings, nil
}

func (a *AdminRepo) loadSettings() (*adminConfig, error) {
	data, err := readFile(a.worktreeFS, "config.yml")
	if err != nil {
		return nil, err
	}

	c := &adminConfig{}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}

	// Now that we have a config, we need to loop through it and expand all
	// users.
	for name := range c.Groups {
		// Replace each of the groups with their expanded versions. This means
		// any future accesses won't need to recurse and so we can ignore the
		// error.
		expanded, err := groupMembers(c.Groups, name, nil)
		if err != nil {
			return nil, err
		}
		c.Groups[name] = expanded
	}

	if c.Repos == nil {
		c.Repos = make(map[string]repoConfig)
	}
	for repoKey, oldRepo := range c.Repos {
		newRepo := oldRepo
		newRepo.Write = expandGroups(c.Groups, newRepo.Write)
		newRepo.Read = expandGroups(c.Groups, newRepo.Read)
		c.Repos[repoKey] = newRepo
	}

	if c.Orgs == nil {
		c.Orgs = make(map[string]orgConfig)
	}
	for orgKey, oldOrg := range c.Orgs {
		newOrg := oldOrg
		newOrg.Admin = expandGroups(c.Groups, newOrg.Admin)
		newOrg.Write = expandGroups(c.Groups, newOrg.Write)
		newOrg.Read = expandGroups(c.Groups, newOrg.Read)

		if newOrg.Repos == nil {
			newOrg.Repos = make(map[string]repoConfig)
		}
		for repoKey, oldRepo := range newOrg.Repos {
			newRepo := oldRepo
			newRepo.Write = expandGroups(c.Groups, newRepo.Write)
			newRepo.Read = expandGroups(c.Groups, newRepo.Read)
			newOrg.Repos[repoKey] = newRepo
		}

		c.Orgs[orgKey] = newOrg
	}

	return c, nil
}

func (a *AdminRepo) ensureSettings() (*adminConfig, error) {
	if a.settings == nil {
		settings, err := a.loadSettings()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to load settings")

			// If we failed to load config, we can update the config with our
			// own sample config.
			err = createFile(a.worktreeFS, "config.yml", []byte(sampleConfig))
			if err != nil {
				return a.settings, err
			}

			_, err = a.worktree.Add("config.yml")
			if err != nil {
				return a.settings, err
			}

			_, err = a.worktree.Commit("Added sample config.yml", &git.CommitOptions{
				Author: newAdminGitSignature(),
			})
			if err != nil {
				return a.settings, err
			}

			// Now that we've saved what should be a valid file, try loading it
			// again.
			settings, err = a.loadSettings()
			if err != nil {
				return a.settings, err
			}
		}

		if settings.Groups == nil || len(settings.Groups["admins"]) == 0 {
			return a.settings, errors.New("No admins defined")
		}

		a.settings = settings
	}

	return a.settings, nil
}

func (a *AdminRepo) loadRSAKey() (*rsa.PrivateKey, error) {
	data, err := readFile(a.worktreeFS, "ssh/id_rsa")
	if err != nil {
		return nil, err
	}

	privateKey, err := gossh.ParseRawPrivateKey(data)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("id_rsa not an RSA key")
	}

	return rsaKey, nil
}

func (a *AdminRepo) loadEd25519Key() (ed25519.PrivateKey, error) {
	data, err := readFile(a.worktreeFS, "ssh/id_ed25519")
	if err != nil {
		return nil, err
	}

	privateKey, err := gossh.ParseRawPrivateKey(data)
	if err != nil {
		return nil, err
	}

	ed25519Key, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("id_ed25519 not an RSA key")
	}

	return ed25519Key, nil
}

// GetServerKeys will return the server's SSH keys
func (a *AdminRepo) GetServerKeys() (keyConfig, error) {
	a.RLock()
	defer a.RUnlock()

	kc := a.keys
	if kc.RSA == nil || kc.Ed25519 == nil {
		return a.keys, errors.New("SSH keys not loaded")
	}

	return kc, nil
}

func (a *AdminRepo) loadServerKeys() keyConfig {
	var err error
	var kc keyConfig

	// If a key doesn't exist, try to load it from a file. If that fails, we'll
	// catch it later and write a new commit with updated keys.
	kc.RSA, err = a.loadRSAKey()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load RSA key")
	}

	kc.Ed25519, err = a.loadEd25519Key()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load ed25519 key")
	}

	return kc
}

func (a *AdminRepo) ensureServerKeys() (keyConfig, error) {
	var err error

	// This function *should* return an error, but because we generate missing
	// keys, it doesn't matter.
	kc := a.loadServerKeys()

	// If either of the keys is still nil, we need to generate them
	if kc.RSA == nil || kc.Ed25519 == nil {
		if kc.RSA == nil {
			log.Warn().Msg("Generating new RSA key")
			kc.RSA, err = generateRSAKey()
			if err != nil {
				return a.keys, err
			}

			rsaBytes := marshalRSAKey(kc.RSA)

			f, err := a.worktreeFS.Create("ssh/id_rsa")
			if err != nil {
				return a.keys, err
			}

			_, err = f.Write(rsaBytes)
			if err != nil {
				return a.keys, err
			}

			_, err = a.worktree.Add("ssh/id_rsa")
			if err != nil {
				return a.keys, err
			}
		}

		if kc.Ed25519 == nil {
			log.Warn().Msg("Generating new ed25519 key")
			kc.Ed25519, err = generateEd25519Key()
			if err != nil {
				return a.keys, err
			}

			ed25519Bytes, err := marshalEd25519Key(kc.Ed25519)
			if err != nil {
				return a.keys, err
			}

			f, err := a.worktreeFS.Create("ssh/id_ed25519")
			if err != nil {
				return a.keys, err
			}

			_, err = f.Write(ed25519Bytes)
			if err != nil {
				return a.keys, err
			}

			_, err = a.worktree.Add("ssh/id_ed25519")
			if err != nil {
				return a.keys, err
			}
		}

		_, err = a.worktree.Commit("Updated ssh keys", &git.CommitOptions{
			Author: newAdminGitSignature(),
		})
		if err != nil {
			return a.keys, err
		}
	}

	// Copy the keys back to the server
	a.keys = kc

	return a.keys, nil
}

func (a *AdminRepo) GetUserFromKey(key ssh.PublicKey) (*User, error) {
	marshaledKey := key.Marshal()

	keys, err := a.GetUserKeys()
	if err != nil {
		return nil, err
	}

	username, ok := keys[string(marshaledKey)]
	if !ok {
		return nil, errors.New("Key does not match a user")
	}

	return a.GetUser(username)
}

func (a *AdminRepo) GetUser(username string) (*User, error) {
	a.RLock()
	defer a.RUnlock()

	if a.users == nil {
		return nil, errors.New("User keys not loaded")
	}

	u, ok := a.users[sanitize(username)]
	if !ok {
		return nil, errors.New("User does not exist")
	}

	return u, nil
}

// GetUserKeys will return a mapping of the marshalled PublicKey to username.
func (a *AdminRepo) GetUserKeys() (map[string]string, error) {
	a.RLock()
	defer a.RUnlock()

	if a.userKeys == nil {
		return nil, errors.New("User keys not loaded")
	}

	return a.userKeys, nil
}

func (a *AdminRepo) loadUsers() (map[string]*User, map[string]string, error) {
	users := make(map[string]*User)
	userKeys := make(map[string]string)

	entries, err := a.worktreeFS.ReadDir("users")
	if err != nil {
		return nil, nil, err
	}

	usersFS, err := a.worktreeFS.Chroot("users")
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".yml") {
			continue
		}

		username := sanitize(filename[:len(filename)-4])

		ulog := log.With().Str("username", username).Logger()

		ulog.Debug().Msg("Found user")

		data, err := readFile(usersFS, filename)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to load user")
			continue
		}

		u := &User{Username: username}
		err = yaml.Unmarshal(data, u)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to load user")
			continue
		}

		// There are two places a user can be defined as an admin.
		if listContains(a.settings.Groups["admins"], u.Username) {
			u.IsAdmin = true
		}

		users[username] = u
		for _, k := range u.PublicKeys {
			userKeys[string(k.Marshal())] = username
		}
	}

	return users, userKeys, nil
}

func (a *AdminRepo) ensureUsers() (map[string]*User, map[string]string, error) {
	if a.users == nil || a.userKeys == nil {
		users, userKeys, err := a.loadUsers()
		if err != nil {
			return nil, nil, err
		}

		a.users = users
		a.userKeys = userKeys
	}

	return a.users, a.userKeys, nil
}
