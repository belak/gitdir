package gitdir

import (
	"errors"
	"path"
	"strings"
)

// RepoType represents the different types of repositories that can be accessed.
type RepoType int

// RepoType defaults to RepoTypeAdmin to make sure that if this is improperly
// set, the only way to access it is by being an admin.
const (
	RepoTypeAdmin RepoType = iota
	RepoTypeOrgConfig
	RepoTypeOrg
	RepoTypeUserConfig
	RepoTypeUser
	RepoTypeTopLevel
)

// RepoLookup represents a repository that has been confirmed in the config and
// the access level the given user has.
type RepoLookup struct {
	Type      RepoType
	PathParts []string
	Access    AccessType
}

// Path returns the full path to this repository on disk. This is relative to
// the gitdir root.
func (repo RepoLookup) Path() string {
	switch repo.Type {
	case RepoTypeAdmin:
		return "admin/admin"
	case RepoTypeOrgConfig:
		return path.Join("admin", "org-"+repo.PathParts[0])
	case RepoTypeOrg:
		return path.Join("orgs", repo.PathParts[0], repo.PathParts[1])
	case RepoTypeUserConfig:
		return path.Join("admin", "user-"+repo.PathParts[0])
	case RepoTypeUser:
		return path.Join("users", repo.PathParts[0], repo.PathParts[1])
	case RepoTypeTopLevel:
		return path.Join("top-level", repo.PathParts[0])
	}

	return "/dev/null"
}

// ErrInvalidRepoFormat is returned when a repo is looked up which cannot
// exist based on the parsed format.
var ErrInvalidRepoFormat = errors.New("invalid repo format")

// ErrRepoDoesNotExist is returned when a repo is looked up which cannot exist
// based on the config.
var ErrRepoDoesNotExist = errors.New("repo does not exist")

// LookupRepoAccess checks to see if path points to a valid repo and attaches
// the access level this user has on that repository.
func (serv *Server) LookupRepoAccess(user *User, path string) (*RepoLookup, error) {
	serv.lock.RLock()
	defer serv.lock.RUnlock()

	repo, err := serv.lookupRepo(path)
	if err != nil {
		return nil, err
	}

	repo.Access = serv.checkUserRepoAccess(user, repo)

	return repo, nil
}

func (serv *Server) lookupRepo(path string) (*RepoLookup, error) {
	// Chop off .git for looking up the repo
	path = strings.TrimSuffix(path, ".git")

	if path == "admin" {
		return &RepoLookup{
			Type:      RepoTypeAdmin,
			PathParts: []string{"admin", "admin"},
		}, nil
	}

	if strings.HasPrefix(path, serv.settings.Options.OrgPrefix) {
		return serv.lookupOrgRepo(strings.TrimPrefix(path, serv.settings.Options.OrgPrefix))
	}

	if strings.HasPrefix(path, serv.settings.Options.UserPrefix) {
		return serv.lookupUserRepo(strings.TrimPrefix(path, serv.settings.Options.UserPrefix))
	}

	return serv.lookupTopLevelRepo(path)
}

func (serv *Server) lookupOrgRepo(path string) (*RepoLookup, error) {
	ret := &RepoLookup{
		PathParts: strings.Split(path, "/"),
	}

	// If there was no org specified or it has more than 2 slashes, it's an
	// invalid repo.
	if len(ret.PathParts) == 0 || len(ret.PathParts) > 2 {
		return nil, ErrInvalidRepoFormat
	}

	// If the org doesn't exist, nobody has access.
	org, ok := serv.settings.Orgs[ret.PathParts[0]]
	if !ok {
		return nil, ErrRepoDoesNotExist
	}

	if len(ret.PathParts) == 1 {
		ret.Type = RepoTypeOrgConfig
		return ret, nil
	}

	// Past this point, it has to be an org repo.
	ret.Type = RepoTypeOrg

	// If implicit repos are enabled, it exists no matter what.
	if serv.settings.Options.ImplicitRepos {
		return ret, nil
	}

	// If the repo explicitly exists in the admin config, this repo exists.
	_, ok = org.Repos[ret.PathParts[1]]
	if ok {
		return ret, nil
	}

	return nil, ErrRepoDoesNotExist
}

func (serv *Server) lookupUserRepo(path string) (*RepoLookup, error) {
	ret := &RepoLookup{
		PathParts: strings.Split(path, "/"),
	}

	// If there was no user specified or it has more than 2 slashes, it's an
	// invalid repo.
	if len(ret.PathParts) == 0 || len(ret.PathParts) > 2 {
		return nil, ErrInvalidRepoFormat
	}

	// If the user doesn't exist, nobody has access.
	user, ok := serv.settings.Users[ret.PathParts[0]]
	if !ok {
		return nil, ErrRepoDoesNotExist
	}

	if len(ret.PathParts) == 1 {
		ret.Type = RepoTypeUserConfig
		return ret, nil
	}

	// Past this point, it has to be an org repo.
	ret.Type = RepoTypeUser

	// If implicit repos are enabled, it exists no matter what.
	if serv.settings.Options.ImplicitRepos {
		return ret, nil
	}

	// If the repo explicitly exists in the admin config, this repo exists.
	_, ok = user.Repos[ret.PathParts[1]]
	if ok {
		return ret, nil
	}

	return nil, ErrRepoDoesNotExist
}

func (serv *Server) lookupTopLevelRepo(path string) (*RepoLookup, error) {
	repoPath := strings.Split(path, "/")
	if len(repoPath) != 1 {
		return nil, ErrInvalidRepoFormat
	}

	ret := &RepoLookup{
		Type:      RepoTypeTopLevel,
		PathParts: repoPath,
	}

	// If implicit repos are enabled, it exists no matter what.
	if serv.settings.Options.ImplicitRepos {
		return ret, nil
	}

	_, ok := serv.settings.Repos[repoPath[0]]
	if ok {
		return ret, nil
	}

	return nil, ErrInvalidRepoFormat
}
