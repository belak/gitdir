package models

import (
	"bytes"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// PublicKey is a wrapper around gossh.PublicKey to also store the comment.
// Note when using that pk.Marshal() handles the wire format, not the
// authorized keys format.
type PublicKey struct {
	ssh.PublicKey

	Comment string
}

// ParsePublicKey will return a PublicKey from the given data.
func ParsePublicKey(data []byte) (*PublicKey, error) {
	var err error

	var pk PublicKey

	pk.PublicKey, pk.Comment, _, _, err = ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, err
	}

	return &pk, nil
}

// UnmarshalYAML implements yaml.Unmarshaler.UnmarshalYAML
func (pk *PublicKey) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var rawData string

	err := unmarshal(&rawData)
	if err != nil {
		return err
	}

	pk.PublicKey, pk.Comment, _, _, err = ssh.ParseAuthorizedKey([]byte(rawData))
	if err != nil {
		return err
	}

	return nil
}

// String implements fmt.Stringer
func (pk *PublicKey) String() string {
	return pk.MarshalAuthorizedKey()
}

// RawMarshalAuthorizedKey converts a key to the authorized keys format,
// without the comment.
func (pk *PublicKey) RawMarshalAuthorizedKey() string {
	if pk == nil || pk.PublicKey == nil {
		return ""
	}

	return string(bytes.TrimSpace(gossh.MarshalAuthorizedKey(pk)))
}

// MarshalAuthorizedKey converts a key to the authorized keys format,
// including a comment.
func (pk *PublicKey) MarshalAuthorizedKey() string {
	if pk == nil {
		return ""
	}

	key := pk.RawMarshalAuthorizedKey()

	if pk.Comment != "" {
		return key + " " + pk.Comment
	}

	return key
}
