package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/belak/go-gitdir/models"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type PublicKeyValue struct {
	PK *models.PublicKey
}

func (pkv PublicKeyValue) String() string {
	if pkv.PK == nil || pkv.PK.PublicKey == nil {
		home, err := homedir.Dir()
		if err != nil {
			panic(err)
		}

		return filepath.Join(home, ".ssh", "id_rsa.pub")
	}

	return ""
}

func (pkv PublicKeyValue) Set(filename string) error {
	fmt.Println("Set:", filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	pk, err := models.ParsePublicKey(data)
	if err != nil {
		return err
	}

	*pkv.PK = *pk

	return nil
}

func (pkv PublicKeyValue) Type() string {
	return "public-key"
}

func initCmd() *cobra.Command {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	var (
		pk = &models.PublicKey{}
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Init the admin repo",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_, err := NewEnvConfig()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to load base config")
			}

			fmt.Println(args)
			fmt.Println(pk.String())
		},
	}

	flags := cmd.Flags()
	pkFlag := flags.VarPF(PublicKeyValue{pk}, "public-key", "", "Admin user's public key file")
	pkFlag.DefValue = filepath.Join(home, ".ssh", "id_rsa.pub")
	pkFlag.NoOptDefVal = pkFlag.DefValue

	return cmd
}
