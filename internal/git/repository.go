package git

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	billy "gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/util"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
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
	RepoFS     *filesystem.Storage
	Worktree   *git.Worktree
	WorktreeFS billy.Filesystem
}

// Open will open a repository if it exists.
func Open(baseFS billy.Filesystem, path string) (*Repository, error) {
	// This lets us sanitize the path and ensure it always has .git on the end.
	path = strings.TrimSuffix(path, ".git") + ".git"

	fs, err := baseFS.Chroot(path)
	if err != nil {
		return nil, err
	}

	repoFS := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())

	// TODO: this probably shouldn't be memfs.
	worktreeFS := memfs.New()

	repo, err := git.Open(repoFS, worktreeFS)
	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	return &Repository{
		Repo:       repo,
		RepoFS:     repoFS,
		Worktree:   worktree,
		WorktreeFS: worktreeFS,
	}, nil
}

// EnsureRepo will open a repository if it exists and try to create it if it
// doesn't.
func EnsureRepo(baseFS billy.Filesystem, path string) (*Repository, error) {
	// This lets us sanitize the path and ensure it always has .git on the end.
	path = strings.TrimSuffix(path, ".git") + ".git"

	if !dirExists(baseFS, path) {
		oldPath := strings.TrimSuffix(path, ".git")

		// If the old dir exists, rename it. Otherwize, init the repo.
		if dirExists(baseFS, oldPath) {
			err := baseFS.Rename(oldPath, path)
			if err != nil {
				return nil, err
			}
		} else {
			fs, err := baseFS.Chroot(path)
			if err != nil {
				return nil, err
			}

			repoFS := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())

			// Init the repo without a worktree so it's a bare repo.
			_, err = git.Init(repoFS, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	repo, err := Open(baseFS, path)
	if err != nil {
		return nil, err
	}

	err = ensureHooks(repo.RepoFS.Filesystem())
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// Checkout will checkout the given hash to the worktreeFS. If an empty string
// is given, we checkout master.
func (r *Repository) Checkout(hash string) error {
	opts := &git.CheckoutOptions{
		Force: true,
	}

	if hash != "" {
		opts.Hash = plumbing.NewHash(hash)
	}

	err := r.Worktree.Checkout(opts)

	// It's fine to ignore ErrReferenceNotFound because that means this is a
	// repo without any commits which doesn't matter for our use cases.
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return err
	}

	return nil
}

func ensureHooks(fs billy.Filesystem) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		err := fs.MkdirAll("hooks/"+hook.Name+".d", 0777)
		if err != nil {
			return err
		}

		// Write the gitdir hook
		err = writeIfDifferent(
			fs,
			"hooks/"+hook.Name+".d/gitdir",
			[]byte(fmt.Sprintf(hook.GitdirHookTemplate, exe)),
		)
		if err != nil {
			return err
		}

		// Write out the actual hook
		//
		// TODO: warn when this would clobber a file
		err = writeIfDifferent(fs, "hooks/"+hook.Name, []byte(hookTemplate))
		if err != nil {
			return err
		}
	}

	return nil
}

func writeIfDifferent(fs billy.Basic, path string, data []byte) error {
	var oldData []byte

	f, err := fs.Open(path)
	if err == nil {
		defer f.Close()
		oldData, _ = ioutil.ReadAll(f)
	}

	// Quick check to avoid unneeded writes
	if !bytes.Equal(oldData, data) {
		return util.WriteFile(fs, path, data, 0777)
	}

	return nil
}

// hookTemplate is based on a combination of sources, but allows us to run
// multiple hooks from a directory (all of them will always be run) and only
// fail if at least one of them failed. This should support every type of git
// hook, as it proxies both stdin and arguments.
var hookTemplate = `#!/usr/bin/env sh
set -e
test -n "${GIT_DIR}" || exit 1

stdin=$(cat)
hookname=$(basename $0)
exitcodes=""

for hook in ${GIT_DIR}/hooks/${hookname}.d/*; do
	# Avoid running non-executable hooks
	test -x "${hook}" || continue

	# Run the actual hook
	echo "${stdin}" | "${hook}" "$@"

	# Store the exit code for later use
	exitcodes="${exitcodes} $?"
done

# Exit on the first non-zero exit code.
for code in ${exitcodes}; do
	test ${code} -eq 0 || exit ${i}
done

exit 0
`

var hooks = []struct {
	Name               string
	GitdirHookTemplate string
}{
	{
		Name: "pre-receive",
		GitdirHookTemplate: `#!/usr/bin/env sh

if [ -z "$GITDIR_BASE_DIR" ]; then
	echo "Warning: GITDIR_BASE_DIR not defined. Skipping hooks."
	exit 0
fi

%q hook pre-receive
`,
	},
	{
		Name: "update",
		GitdirHookTemplate: `#!/usr/bin/env sh

if [ -z "$GITDIR_BASE_DIR" ]; then
	echo "Warning: GITDIR_BASE_DIR not defined. Skipping hooks."
	exit 0
fi

%q hook update $1 $2 $3
`,
	},
	{
		Name: "post-receive",
		GitdirHookTemplate: `#!/usr/bin/env sh

if [ -z "$GITDIR_BASE_DIR" ]; then
	echo "Warning: GITDIR_BASE_DIR not defined. Skipping hooks."
	exit 0
fi

%q hook post-receive
`,
	},
}
