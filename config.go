package gitdir

import (
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"

	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

// Config stores all the server-level settings. These cannot be changed at
// runtime.
type Config struct {
	BindAddr  string
	BasePath  string
	LogFormat string
	LogDebug  bool
}

// DefaultConfig is used as the base config.
var DefaultConfig = &Config{
	BindAddr:  ":2222",
	BasePath:  "./tmp",
	LogFormat: "json",
}

// TODO: clean up nolint here
func (serv *Server) reloadInternal() error { //nolint:funlen
	// Clear configs for users and orgs. We will load them in later, after the
	// main config is loaded.
	serv.users = make(map[string]*models.UserConfig)
	serv.orgs = make(map[string]*models.OrgConfig)
	serv.publicKeys = make(map[string]string)

	// Open the admin repo to the latest commit
	adminRepo, err := git.EnsureRepo("admin/admin", true)
	if err != nil {
		return err
	}

	// Do what we can to ensure we have a config that can be loaded.
	serv.ensureConfig(adminRepo)
	serv.ensureEd25519Key(adminRepo)
	serv.ensureRSAKey(adminRepo)

	err = serv.reloadConfig(adminRepo)
	if err != nil {
		return err
	}

	err = serv.reloadKeys(adminRepo)
	if err != nil {
		return err
	}

	// Reload users
	err = serv.reloadUsers()
	if err != nil {
		return err
	}

	// Reload orgs
	err = serv.reloadOrgs()
	if err != nil {
		return err
	}

	// Load all user keys into the server's public keys. We load from user repos
	// before admin repos so anything defined in the admin repos will override
	// the users.
	for username, user := range serv.users {
		for _, pk := range user.Keys {
			serv.publicKeys[pk.RawMarshalAuthorizedKey()] = username
		}
	}

	for username, user := range serv.settings.Users {
		for _, pk := range user.Keys {
			serv.publicKeys[pk.RawMarshalAuthorizedKey()] = username
		}
	}

	// We actually only commit at the very end, after everything has been
	// loaded. This ensures we have a valid config.
	status, err := adminRepo.Worktree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		err = adminRepo.Commit("Updated config", nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serv *Server) ensureConfig(adminRepo *git.Repository) {
	err := adminRepo.UpdateFile("config.yml", ensureSampleConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to update config.yml")
	}
}

func (serv *Server) ensureEd25519Key(adminRepo *git.Repository) {
	err := adminRepo.UpdateFile("ssh/id_ed25519", func(data []byte) ([]byte, error) {
		if data != nil {
			return data, nil
		}

		pk, err := models.GenerateEd25519PrivateKey()
		if err != nil {
			return nil, err
		}

		return pk.MarshalPrivateKey()
	})

	if err != nil {
		log.Warn().Err(err).Msg("Failed to update ssh/id_ed25519")
	}
}

func (serv *Server) ensureRSAKey(adminRepo *git.Repository) {
	err := adminRepo.UpdateFile("ssh/id_rsa", func(data []byte) ([]byte, error) {
		if data != nil {
			return data, nil
		}

		pk, err := models.GenerateRSAPrivateKey()
		if err != nil {
			return nil, err
		}

		return pk.MarshalPrivateKey()
	})

	if err != nil {
		log.Warn().Err(err).Msg("Failed to update ssh/id_rsa")
	}
}

func (serv *Server) reloadConfig(r *git.Repository) error {
	configData, err := r.GetFile("config.yml")
	if err != nil {
		return err
	}

	serv.settings, err = models.ParseAdminConfig(configData)

	return err
}

func (serv *Server) reloadKeys(r *git.Repository) error {
	var pks []models.PrivateKey

	// Load the ed25519 key
	keyData, err := r.GetFile("ssh/id_ed25519")
	if err != nil {
		return err
	}

	pk, err := models.ParseEd25519PrivateKey(keyData)
	if err != nil {
		return err
	}

	pks = append(pks, pk)

	// Load the RSA key
	keyData, err = r.GetFile("ssh/id_rsa")
	if err != nil {
		return err
	}

	pk, err = models.ParseRSAPrivateKey(keyData)
	if err != nil {
		return err
	}

	pks = append(pks, pk)

	// Actually add the ssh keys to the server
	for _, key := range pks {
		signer, err := gossh.NewSignerFromSigner(key)
		if err != nil {
			return err
		}

		serv.s.AddHostKey(signer)
	}

	return nil
}

func (serv *Server) reloadUsers() error {
	for username := range serv.settings.Users {
		userRepo, err := git.EnsureRepo("admin/user-"+username, true)
		if err != nil {
			return err
		}

		data, err := userRepo.GetFile("config.yml")
		if err != nil {
			return err
		}

		userConfig, err := models.ParseUserConfig(data)
		if err != nil {
			return err
		}

		serv.users[username] = userConfig
	}

	return nil
}

func (serv *Server) reloadOrgs() error {
	for orgName := range serv.settings.Orgs {
		orgRepo, err := git.EnsureRepo("admin/org-"+orgName, true)
		if err != nil {
			return err
		}

		data, err := orgRepo.GetFile("config.yml")
		if err != nil {
			return err
		}

		orgConfig, err := models.ParseOrgConfig(data)
		if err != nil {
			return err
		}

		serv.orgs[orgName] = orgConfig
	}

	return nil
}
