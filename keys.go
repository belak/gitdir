package main

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type PublicKey struct {
	ssh.PublicKey
	comment string
}

// Implement loading from a file. This is used by the cli package.
func (pk *PublicKey) Set(value string) error {
	var err error

	rawData, err := ioutil.ReadFile(value)
	if err != nil {
		return err
	}

	pk.PublicKey, pk.comment, _, _, err = ssh.ParseAuthorizedKey(rawData)
	if err != nil {
		return err
	}

	return nil
}

// TODO: convert to []byte
func (pk *PublicKey) RawMarshalAuthorizedKey() string {
	if pk == nil || pk.PublicKey == nil {
		return ""
	}
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(pk)))
}

func (pk *PublicKey) MarshalAuthorizedKey() string {
	key := pk.RawMarshalAuthorizedKey()
	if pk.comment != "" {
		return key + " " + pk.comment
	}
	return key
}

// String makes it much easier to support yaml
func (pk *PublicKey) String() string {
	return pk.MarshalAuthorizedKey()
}

// Implement loading from yaml files
func (pk *PublicKey) UnmarshalYAML(unmarshal func(v interface{}) error) error {
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

type PrivateKey interface {
	crypto.Signer

	MarshalPrivateKey() ([]byte, error)
}

type ed25519PrivateKey struct {
	ed25519.PrivateKey
}

func ParseEd25519Key(data []byte) (PrivateKey, error) {
	privateKey, err := gossh.ParseRawPrivateKey(data)
	if err != nil {
		return nil, err
	}

	ed25519Key, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("id_ed25519 not an RSA key")
	}

	return &ed25519PrivateKey{ed25519Key}, nil
}

func GenerateEd25519Key() (PrivateKey, error) {
	_, pk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &ed25519PrivateKey{pk}, err
}

func (pk *ed25519PrivateKey) MarshalPrivateKey() ([]byte, error) {
	// Get ASN.1 DER format
	privDER, err := x509.MarshalPKCS8PrivateKey(pk.PrivateKey)
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

type rsaPrivateKey struct {
	*rsa.PrivateKey
}

func ParseRSAKey(data []byte) (PrivateKey, error) {
	privateKey, err := gossh.ParseRawPrivateKey(data)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}

	return &rsaPrivateKey{rsaKey}, nil
}

func GenerateRSAKey() (PrivateKey, error) {
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

	return &rsaPrivateKey{privateKey}, nil
}

func (pk *rsaPrivateKey) MarshalPrivateKey() ([]byte, error) {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(pk.PrivateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM, nil
}
