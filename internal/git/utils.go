package git

import (
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func newAdminGitSignature() *object.Signature {
	return &object.Signature{
		Name:  "root",
		Email: "root@localhost",
		When:  time.Now(),
	}
}
