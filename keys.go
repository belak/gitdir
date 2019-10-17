package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/gliderlabs/ssh"
)

type publicKey struct {
	ssh.PublicKey
}

func (pk *publicKey) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var rawData string
	err := unmarshal(&rawData)
	if err != nil {
		return err
	}

	pk.PublicKey, _, _, _, err = ssh.ParseAuthorizedKey([]byte(rawData))
	if err != nil {
		return err
	}

	return nil
}

func generateEd25519Key() (ed25519.PrivateKey, error) {
	_, pk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return pk, err
}

func marshalEd25519Key(pk ed25519.PrivateKey) ([]byte, error) {
	// Get ASN.1 DER format
	privDER, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		return nil, err
	}

	// pem.Block
	privBlock := pem.Block{
		Type:    "PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM, nil
}

func generateRSAKey() (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func marshalRSAKey(pk *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(pk)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}
