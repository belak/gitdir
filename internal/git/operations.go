package git

import (
	"io/ioutil"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetFile is a convenience method to get the contents of a file in the repo.
func (r *Repository) GetFile(filename string) ([]byte, error) {
	f, err := r.WorktreeFS.Open(filename)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

// FileExists returns true if the given path exists and is a regular file in the
// repository worktree.
func (r *Repository) FileExists(filename string) bool {
	stat, err := r.WorktreeFS.Stat(filename)
	if err != nil {
		return false
	}

	return stat.Mode().IsRegular()
}

// DirExists returns true if the given path exists and is a directory in the
// repository worktree.
func (r *Repository) DirExists(filename string) bool {
	stat, err := r.WorktreeFS.Stat(filename)
	if err != nil {
		return false
	}

	return stat.Mode().IsDir()
}

// CreateFile is a convenience method to set the contents of a file in the
// repo and stage it.
func (r *Repository) CreateFile(filename string, data []byte) error {
	f, err := r.WorktreeFS.Create(filename)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	_, err = r.Worktree.Add(filename)

	return err
}

// UpdateFile is a convenience method to get the contents of a file, pass it
// to a callback, write the contents back to the file if no error was
// returned, and stage the file.
func (r *Repository) UpdateFile(filename string, cb func([]byte) ([]byte, error)) error {
	data, _ := r.GetFile(filename)

	data, err := cb(data)
	if err != nil {
		return err
	}

	return r.CreateFile(filename, data)
}

// Commit is a convenience method to make working with the worktree a little
// bit easier.
func (r *Repository) Commit(msg string, author *object.Signature) error {
	if author == nil {
		author = newAdminGitSignature()
	}

	_, err := r.Worktree.Commit(msg, &git.CommitOptions{
		Author: author,
	})

	return err
}
