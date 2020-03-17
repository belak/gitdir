package git

import (
	"time"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Hash = plumbing.Hash

var ZeroHash Hash = plumbing.ZeroHash

func NewHash(hash string) Hash {
	return plumbing.NewHash(hash)
}

func newAdminGitSignature() *object.Signature {
	return &object.Signature{
		Name:  "root",
		Email: "root@localhost",
		When:  time.Now(),
	}
}

func dirExists(fs billy.Filesystem, path string) bool {
	info, err := fs.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
