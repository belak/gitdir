package git

import (
	"time"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

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
