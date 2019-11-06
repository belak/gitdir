package git

import (
	"time"

	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
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