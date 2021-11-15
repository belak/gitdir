package models

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	gossh "golang.org/x/crypto/ssh"
)

// PrivateKey is a wrapper to make dealing with private keys easier to deal
// with.
type PrivateKey interface {
	crypto.Signer

	MarshalPrivateKey() ([]byte, error)
}

type ed25519PrivateKey struct {
	ed25519.PrivateKey
}

// ParseEd25519PrivateKey parses an ed25519 private key.
func ParseEd25519PrivateKey(data []byte) (PrivateKey, error) {
	privateKey, err := gossh.ParseRawPrivateKey(data)
	if err != nil {
		return nil, err
	}

	// Try loading as an external key, fall back to internal key. This *should*
	// fix issues with incompatible versions.
	ed25519Key, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("not an ed25519 key")
	}

	return &ed25519PrivateKey{ed25519Key}, nil
}

// GenerateEd25519PrivateKey generates a new ed25519 private key.
func GenerateEd25519PrivateKey() (PrivateKey, error) {
	_, pk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &ed25519PrivateKey{pk}, err
}

// MarshalPrivateKey implements PrivateKey.MarshalPrivateKey.
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

// ParseRSAPrivateKey parses an RSA private key.
func ParseRSAPrivateKey(data []byte) (PrivateKey, error) {
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

// GenerateRSAPrivateKey generates a new RSA private key of size 4096.
func GenerateRSAPrivateKey() (PrivateKey, error) {
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

// MarshalPrivateKey implements PrivateKey.MarshalPrivateKey.
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
