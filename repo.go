package main

import (
	"errors"
	"path"
	"strings"
)

type RepoType int

const (
	// panic()
	RepoTypeUnknown RepoType = iota
	// admin
	RepoTypeAdmin
	// repo
	RepoTypeTopLevel
	// ~user
	RepoTypeUserConfig
	// ~user/repo
	RepoTypeUser
	// @org
	RepoTypeOrgConfig
	// @org/repo
	RepoTypeOrg
)

// RepoLookup represents a repo query. This is a simple type used to
type RepoLookup struct {
	Type RepoType
	Dir  string
	Name string
	Path string
}

func ParseRepo(c *Config, pathname string) (*RepoLookup, error) {
	r, err := parseRepoInternal(c, pathname)
	if err != nil {
		return nil, err
	}

	r.Path, err = r.buildPath(c)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func parseRepoInternal(c *Config, pathname string) (*RepoLookup, error) {
	r := &RepoLookup{
		Type: RepoTypeUnknown,
	}

	path := strings.Split(pathname, "/")

	// All config types are technically in the root dir
	if len(path) == 1 {
		dir := path[0]
		if path[0] == "admin" {
			r.Type = RepoTypeAdmin
		} else if strings.HasPrefix(dir, c.UserPrefix) {
			r.Type = RepoTypeUserConfig
			r.Name = dir[len(c.UserPrefix):]
		} else if strings.HasPrefix(path[0], c.OrgPrefix) {
			r.Type = RepoTypeOrgConfig
			r.Name = dir[len(c.OrgPrefix):]
		} else {
			r.Type = RepoTypeTopLevel
			r.Name = dir
		}
		return r, nil
	}

	if len(path) != 2 {
		return nil, errors.New("Invalid repo format")
	}

	dir := path[0]
	name := path[1]

	if strings.HasPrefix(dir, c.UserPrefix) {
		r.Type = RepoTypeUser
		r.Dir = dir[len(c.UserPrefix):]
		r.Name = name
		return r, nil
	}

	if strings.HasPrefix(dir, c.OrgPrefix) {
		r.Type = RepoTypeOrg
		r.Dir = dir[len(c.OrgPrefix):]
		r.Name = name
		return r, nil
	}

	return nil, errors.New("Invalid repo format")
}

func (serv *server) LookupRepo(repoPath string, u *User, access accessType) (*RepoLookup, error) {
	serv.repo.RLock()
	defer serv.repo.RUnlock()

	lookup, err := ParseRepo(serv.c, repoPath)
	if err != nil {
		return nil, err
	}

	if !serv.repo.lookupIsValid(lookup) {
		return nil, errors.New("Repo does not exist")
	}

	if !serv.repo.settings.UserHasRepoAccess(u, lookup, access) {
		return nil, errors.New("Permission denied")
	}

	_, err = EnsureRepo(lookup.Path)
	if err != nil {
		return nil, err
	}

	return lookup, nil
}

func (a *AdminRepo) lookupIsValid(r *RepoLookup) bool {
	switch r.Type {
	case RepoTypeAdmin:
		// An admin repo is always valid
		return true
	case RepoTypeTopLevel:
		// A top level repo is valid if the repo is defined.
		_, ok := a.settings.Repos[r.Name]
		return ok
	case RepoTypeUserConfig:
		// A user config repo is valid if the user is defined.
		_, ok := a.users[r.Name]
		return ok
	case RepoTypeUser:
		// User repos are valid if user repos are enabled and the user is defined.
		if !a.settings.Options.UserRepos {
			return false
		}
		// TODO: users should be able to define these in their user config
		_, ok := a.users[r.Dir]
		return ok
	case RepoTypeOrgConfig:
		// An org config repo is valid if the org is defined.
		_, ok := a.settings.Orgs[r.Name]
		return ok
	case RepoTypeOrg:
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

func (r RepoLookup) buildPath(c *Config) (string, error) {
	switch r.Type {
	case RepoTypeAdmin:
		// Admin repo is a static path
		return path.Join(c.BasePath, "admin", "admin"), nil
	case RepoTypeTopLevel:
		return path.Join(c.BasePath, "top-level", r.Name), nil
	case RepoTypeUserConfig:
		return path.Join(c.BasePath, "admin", "user-"+r.Name), nil
	case RepoTypeUser:
		return path.Join(c.BasePath, "users", r.Dir, r.Name), nil
	case RepoTypeOrgConfig:
		return path.Join(c.BasePath, "admin", "org-"+r.Name), nil
	case RepoTypeOrg:
		return path.Join(c.BasePath, "orgs", r.Dir, r.Name), nil
	}

	return "", errors.New("Unsupported repo type")
}
