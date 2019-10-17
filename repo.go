package main

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

	git "github.com/libgit2/git2go"
)

type repoType int

const (
	// admin
	repoTypeAdmin repoType = iota
	// repo
	repoTypeTopLevel
	// ~user
	repoTypeUserConfig
	// ~user/repo
	repoTypeUser
	// @org
	repoTypeOrgConfig
	// @org/repo
	repoTypeOrg
	// panic()
	repoTypeUnknown
)

type repo struct {
	Dir  string
	Name string
	Type repoType

	Path string
}

func EnsureRepo(c *Config, a *AdminRepo, pathname string) (repo, error) {
	r, err := LookupRepo(c, pathname)
	if err != nil {
		return r, err
	}

	if a.RepoIsValid(r) {
		_, err = r.Ensure()
		if err != nil {
			return r, err
		}
	}

	return r, nil
}

func LookupRepo(c *Config, pathname string) (repo, error) {
	r, err := parseRepo(c, pathname)
	if err != nil {
		return r, err
	}

	repoPath, err := r.buildPath()
	if err != nil {
		return r, err
	}

	r.Path = path.Join(c.BasePath, filepath.FromSlash(repoPath))

	return r, nil
}

func (a *AdminRepo) RepoIsValid(r repo) bool {
	a.RLock()
	defer a.RUnlock()

	switch r.Type {
	case repoTypeAdmin:
		// An admin repo is always valid
		return true
	case repoTypeTopLevel:
		// A top level repo is valid if the repo is defined.
		_, ok := a.settings.Repos[r.Name]
		return ok
	case repoTypeUserConfig:
		// A user config repo is valid if the user is defined.
		_, ok := a.users[r.Name]
		return ok
	case repoTypeUser:
		// User repos are valid if user repos are enabled and the user is defined.
		if !a.settings.Options.UserRepos {
			return false
		}
		// TODO: users should be able to define these in their user config
		_, ok := a.users[r.Dir]
		return ok
	case repoTypeOrgConfig:
		// An org config repo is valid if the org is defined.
		_, ok := a.settings.Orgs[r.Name]
		return ok
	case repoTypeOrg:
		// An org config repo is valid if the org  isdefined and the org repo is
		// defined.
		org, ok := a.settings.Orgs[r.Dir]
		if !ok {
			return false
		}
		// TODO: org admins should be able to define these in their org config
		_, ok = org.Repos[r.Name]
		return ok
	default:
		return false
	}
}

func parseRepo(c *Config, pathname string) (repo, error) {
	r := repo{
		Dir:  path.Dir(pathname),
		Name: path.Base(pathname),
		Type: repoTypeUnknown,
	}

	if strings.Contains(r.Dir, "/") {
		return r, errors.New("Invalid repo format")
	}

	// All config types are technically in the root dir
	if r.Dir == "." {
		if r.Name == "admin" {
			r.Type = repoTypeAdmin
		} else if strings.HasPrefix(r.Name, c.UserPrefix) {
			r.Name = r.Name[len(c.UserPrefix):]
			r.Type = repoTypeUserConfig
		} else if strings.HasPrefix(r.Name, c.OrgPrefix) {
			r.Name = r.Name[len(c.OrgPrefix):]
			r.Type = repoTypeOrgConfig
		} else {
			r.Type = repoTypeTopLevel
		}
		return r, nil
	}

	if strings.HasPrefix(r.Dir, c.UserPrefix) {
		r.Dir = r.Dir[len(c.UserPrefix):]
		r.Type = repoTypeUser
		return r, nil
	}

	if strings.HasPrefix(r.Dir, c.OrgPrefix) {
		r.Dir = r.Dir[len(c.OrgPrefix):]
		r.Type = repoTypeOrg
		return r, nil
	}

	return r, errors.New("Invalid repo format")
}

func (r repo) Open() (*Repo, error) {
	repo, err := git.OpenRepositoryExtended(r.Path, repoOpenFlags, "")
	return &Repo{repo}, err
}

func (r repo) Exists() (bool, error) {
	_, err := r.Open()
	if err != nil {
		if gitError, ok := err.(*git.GitError); ok {
			if gitError.Class != git.ErrClassOs || gitError.Code != git.ErrNotFound {
				return false, err
			}

			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r repo) Create() (*Repo, error) {
	repo, err := git.InitRepository(r.Path, true)
	if err != nil {
		return nil, err
	}
	return &Repo{repo}, nil
}

func (r repo) Ensure() (*Repo, error) {
	ok, err := r.Exists()
	if err != nil {
		return nil, err
	}

	if !ok {
		return r.Create()
	}

	return r.Open()
}

func (r repo) buildPath() (string, error) {
	// Mapping of repo type to path. Note that because user and org configs are
	// also in the admin directory, it's technically possible for an admin to
	// clone them using org-orgname or user-username. This is fine since admins
	// are considered super-admins.
	switch r.Type {
	case repoTypeAdmin:
		return path.Join("admin", r.Name), nil
	case repoTypeTopLevel:
		return path.Join("top-level", r.Name), nil
	case repoTypeUserConfig:
		return path.Join("admin", "user-"+r.Name), nil
	case repoTypeUser:
		return path.Join("users", r.Dir, r.Name), nil
	case repoTypeOrg:
		return path.Join("admin", "org-"+r.Name), nil
	case repoTypeOrgConfig:
		return path.Join("orgs", r.Dir, r.Name), nil
	}

	return "", errors.New("Unsupported repo type")
}
