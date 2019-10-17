package main

import (
	"errors"
	"strings"

	git "github.com/libgit2/git2go"
)

const repoOpenFlags = git.RepositoryOpenNoSearch | git.RepositoryOpenBare

// Repo is a wrapper around a git repository to make a number of common
// operations more convenient.
type Repo struct {
	*git.Repository
}

// EnsureRepo will open a repository if it exists and try to create it if it
// doesn't.
func EnsureRepo(path string) (*Repo, error) {
	repo, err := git.OpenRepositoryExtended(path, repoOpenFlags, "")
	if err != nil {
		gitError, ok := err.(*git.GitError)
		// If it's not a GitError or it's not explicitly an ErrNotFound, we need
		// to error.
		if !ok || gitError.Class != git.ErrClassOs || gitError.Code != git.ErrNotFound {
			return nil, err
		}

		// If the repo explicitly doesn't exist, we need to initialize it.
		repo, err = git.InitRepository(path, true)
		if err != nil {
			return nil, err
		}
	}

	return &Repo{repo}, nil
}

// HeadCommit is a convenience method to get the current Commit that HEAD points
// to.
func (r *Repo) HeadCommit() (*git.Commit, error) {
	head, err := r.Head()
	if err != nil {
		return nil, err
	}

	obj, err := head.Peel(git.ObjectCommit)
	if err != nil {
		return nil, err
	}

	commit, err := obj.AsCommit()
	if err != nil {
		return nil, err
	}

	return commit, nil
}

// HeadTree is a convenience method to get the current Tree that the HEAD commit
// points to.
func (r *Repo) HeadTree() (*git.Tree, error) {
	commit, err := r.HeadCommit()
	if err != nil {
		return nil, err
	}
	return commit.Tree()
}

// GetFile is a convenience method to get the file contents of a given filename
// at the head of the current repo.
func (r *Repo) GetFile(filename string) ([]byte, error) {
	tree, err := r.HeadTree()
	if err != nil {
		return nil, err
	}

	entry, err := tree.EntryByPath(filename)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, errors.New("File missing")
	}

	if entry.Type != git.ObjectBlob {
		return nil, errors.New("File is not a file")
	}

	blob, err := r.LookupBlob(entry.Id)
	if err != nil {
		return nil, err
	}

	return blob.Contents(), nil
}

// CommitBuilder is a wrapper around a git TreeBuilder to make it easier to
// make new commits.
type CommitBuilder struct {
	repo *Repo

	// interface{} in this instance is either map[string]interface{} or []byte.
	// The map represents a directory, bytes represent a file. Nil means delete
	// the file (or directory).
	files map[string]interface{}
}

// CommitBuilder creates a new commit builder for the given repo.
func (r *Repo) CommitBuilder() (*CommitBuilder, error) {
	return &CommitBuilder{
		r,
		make(map[string]interface{}),
	}, nil
}

// AddFile adds a file to be written by the commit builder. Nested paths can be
// used and the CommitBuilder will handle properly creating the nested trees on
// Write. Note that passing in nil as data will cause a file or folder to be
// deleted.
func (b *CommitBuilder) AddFile(filePath string, data []byte) error {
	pathParts := strings.Split(filePath, "/")

	dir := b.files

	for len(pathParts) > 1 {
		// Pop the first item off the stack
		path := pathParts[0]
		pathParts = pathParts[1:]

		rawItem, ok := dir[path]
		if !ok {
			tmp := make(map[string]interface{})
			dir[path] = tmp
			dir = tmp
			continue
		}

		switch item := rawItem.(type) {
		case map[string]interface{}:
			dir = item
		case []byte:
			return errors.New("Overlapping files")
		default:
			return errors.New("Invalid file tree")
		}
	}

	dir[pathParts[0]] = data

	return nil
}

// Write will write all the staged files to disk. Note that this will always
// target the latest commit in HEAD.
func (b *CommitBuilder) Write(message string, author, committer *git.Signature) (*git.Oid, error) {
	if author == nil {
		author = newAdminGitSignature()
	}
	if committer == nil {
		committer = author
	}

	// Start with a nil tree - if there are no commits, we can warn, but we
	// should make the commit anyway.
	var headTree *git.Tree
	var parents []*git.Commit

	unborn, err := b.repo.IsHeadUnborn()
	if err != nil {
		return nil, err
	}

	if !unborn {
		headCommit, err := b.repo.HeadCommit()
		if err != nil {
			return nil, err
		}

		// Add the head commit to the parents
		parents = append(parents, headCommit)

		headTree, err = headCommit.Tree()
		if err != nil {
			return nil, err
		}
	}

	tree, err := b.recursiveBuildTree(headTree, b.files)
	if err != nil {
		return nil, err
	}

	return b.repo.CreateCommit("HEAD", author, committer, message, tree, parents...)
}

func (b *CommitBuilder) recursiveBuildTree(parent *git.Tree, data map[string]interface{}) (*git.Tree, error) {
	var err error
	var builder *git.TreeBuilder

	// Grab the tree builder
	if parent == nil {
		builder, err = b.repo.TreeBuilder()
	} else {
		builder, err = b.repo.TreeBuilderFromTree(parent)
	}
	if err != nil {
		return nil, err
	}

	// Loop through each one of our file entries and create tree entries for
	// them.
	for key, rawItem := range data {
		if rawItem == nil {
			err = builder.Remove(key)
			if err != nil {
				return nil, err
			}
			continue
		}

		switch item := rawItem.(type) {
		case map[string]interface{}:
			var childTree *git.Tree

			// If we got a map we need to recurse
			childEntry := parent.EntryByName(key)

			if childEntry != nil && childEntry.Type == git.ObjectTree {
				childTree, err = b.repo.LookupTree(childEntry.Id)
				if err != nil {
					return nil, err
				}
			}

			tree, err := b.recursiveBuildTree(childTree, item)
			if err != nil {
				return nil, err
			}

			err = builder.Insert(key, tree.AsObject().Id(), git.FilemodeTree)
			if err != nil {
				return nil, err
			}
		case []byte:
			oid, err := b.repo.CreateBlobFromBuffer(item)
			if err != nil {
				return nil, err
			}
			err = builder.Insert(key, oid, git.FilemodeBlob)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("Invalid file tree")
		}
	}

	// Write out the actual tree
	oid, err := builder.Write()
	if err != nil {
		return nil, err
	}

	// Look up the tree we just wrote and return it
	return b.repo.LookupTree(oid)
}
