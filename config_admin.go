package gitdir

import (
	"github.com/belak/go-gitdir/internal/git"
	"github.com/belak/go-gitdir/models"
)

func (c *Config) ensureAdminConfig(repo *git.Repository) error {
	return newMultiError(
		c.ensureAdminConfigYaml(repo),
		c.ensureAdminEd25519Key(repo),
		c.ensureAdminRSAKey(repo),
	)
}

func (c *Config) ensureAdminConfigYaml(repo *git.Repository) error {
	return repo.UpdateFile("config.yml", ensureSampleConfig)
}

func (c *Config) ensureAdminEd25519Key(repo *git.Repository) error {
	return repo.UpdateFile("ssh/id_ed25519", func(data []byte) ([]byte, error) {
		if data != nil {
			return data, nil
		}

		pk, err := models.GenerateEd25519PrivateKey()
		if err != nil {
			return nil, err
		}

		return pk.MarshalPrivateKey()
	})
}

func (c *Config) ensureAdminRSAKey(repo *git.Repository) error {
	return repo.UpdateFile("ssh/id_rsa", func(data []byte) ([]byte, error) {
		if data != nil {
			return data, nil
		}

		pk, err := models.GenerateRSAPrivateKey()
		if err != nil {
			return nil, err
		}

		return pk.MarshalPrivateKey()
	})
}

func (c *Config) loadAdminConfig(adminRepo *git.Repository) error {
	configData, err := adminRepo.GetFile("config.yml")
	if err != nil {
		return err
	}

	adminConfig, err := models.ParseAdminConfig(configData)
	if err != nil {
		return err
	}

	// Load the private keys
	var pks []models.PrivateKey

	keyData, err := adminRepo.GetFile("ssh/id_ed25519")
	if err != nil {
		return err
	}

	pk, err := models.ParseEd25519PrivateKey(keyData)
	if err != nil {
		return err
	}

	pks = append(pks, pk)

	keyData, err = adminRepo.GetFile("ssh/id_rsa")
	if err != nil {
		return err
	}

	pk, err = models.ParseRSAPrivateKey(keyData)
	if err != nil {
		return err
	}

	pks = append(pks, pk)

	// Now that all loading has succeeded, we can replace the values on the
	// config struct.
	c.adminConfig = adminConfig
	c.pks = pks

	return nil
}
