package git

import (
	"io/ioutil"

	"github.com/rs/zerolog/log"
	billy "gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/osfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

// Repository is a fairly lightweight wrapper designed to attach a worktree to a
// git repository so we don't have to muck about with the raw git objects.
//
// We keep the worktree separate from the repo so we can still have a bare repo.
// This also lets us do fun things like keep the worktree in memory if we really
// want to.
type Repository struct {
	Repo       *git.Repository
	Worktree   *git.Worktree
	WorktreeFS billy.Filesystem
}

// EnsureRepo will open a repository if it exists and try to create it if it
// doesn't. runCheckout allows you to skip the checkout of the files in the
// repo.
func EnsureRepo(path string, runCheckout bool) (*Repository, error) {
	// TODO: if .git doesn't exist, but without does, just use that?
	fs := osfs.New(path + ".git")

	// TODO: this probably shouldn't be memfs.
	worktreeFS := memfs.New()

	repoFS := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())

	repo, err := git.Open(repoFS, worktreeFS)
	// If we explicitly got a NotExists error, we should init the repo
	if err == git.ErrRepositoryNotExists {
		log.Warn().Str("repo_path", path).Msg("Repo doesn't exist: creating")

		// Init the repo without a worktree.
		_, err = git.Init(repoFS, nil)
		if err != nil {
			return nil, err
		}

		// Try again to open the repo now that it exists, using a separate
		// worktree fs.
		repo, err = git.Open(repoFS, worktreeFS)
	}

	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	if runCheckout {
		err = worktree.Checkout(&git.CheckoutOptions{
			Force: true,
		})
		// It's fine to ignore ErrReferenceNotFound because that means this is a
		// repo without any commits which doesn't matter for our use cases.
		if err != nil && err != plumbing.ErrReferenceNotFound {
			return nil, err
		}
	}

	return &Repository{
		Repo:       repo,
		Worktree:   worktree,
		WorktreeFS: worktreeFS,
	}, nil
}

// GetFile is a convenience method to get the contents of a file in the repo.
func (wr *Repository) GetFile(filename string) ([]byte, error) {
	f, err := wr.WorktreeFS.Open(filename)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

// FileExists returns true if the given path exists and is a regular file in the
// repository worktree.
func (wr *Repository) FileExists(filename string) bool {
	stat, err := wr.WorktreeFS.Stat(filename)
	if err != nil {
		return false
	}

	return stat.Mode().IsRegular()
}

// DirExists returns true if the given path exists and is a directory in the
// repository worktree.
func (wr *Repository) DirExists(filename string) bool {
	stat, err := wr.WorktreeFS.Stat(filename)
	if err != nil {
		return false
	}

	return stat.Mode().IsDir()
}

// CreateFile is a convenience method to set the contents of a file in the
// repo and stage it.
func (wr *Repository) CreateFile(filename string, data []byte) error {
	f, err := wr.WorktreeFS.Create(filename)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	_, err = wr.Worktree.Add(filename)

	return err
}

// UpdateFile is a convenience method to get the contents of a file, pass it
// to a callback, write the contents back to the file if no error was
// returned, and stage the file.
func (wr *Repository) UpdateFile(filename string, cb func([]byte) ([]byte, error)) error {
	data, _ := wr.GetFile(filename)

	data, err := cb(data)
	if err != nil {
		return err
	}

	return wr.CreateFile(filename, data)
}

// Commit is a convenience method to make working with the worktree a little
// bit easier.
func (wr *Repository) Commit(msg string, author *object.Signature) error {
	if author == nil {
		author = newAdminGitSignature()
	}

	_, err := wr.Worktree.Commit(msg, &git.CommitOptions{
		Author: author,
	})

	return err
}
