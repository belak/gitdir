package main

import "path"

func genericUserHasAccess(rc RepoConfig, u *User, a AccessType) bool {
	if u.IsAdmin {
		return true
	}

	// If a write user is requesting write or below, they can access the repo
	if listContains(rc.Write, u.Username) && a <= AccessTypeWrite {
		return true
	}

	// If a read user is requesting read or below, they can access the repo
	if listContains(rc.Read, u.Username) && a <= AccessTypeRead {
		return true
	}

	return false
}

// RepoLookup represents a query for a repository type.
type RepoLookup interface {
	Path() string
	IsValid(*AdminConfig) bool
	UserHasAccess(*AdminConfig, *User, AccessType) bool
}

// repoLookupAdmin represents the admin repo (admin)
type repoLookupAdmin struct{}

func (rl repoLookupAdmin) Path() string {
	return path.Join("admin", "admin")
}

func (rl repoLookupAdmin) IsValid(c *AdminConfig) bool {
	return true
}

func (rl repoLookupAdmin) UserHasAccess(c *AdminConfig, u *User, a AccessType) bool {
	// Only admin users have access to the admin repo.
	return u.IsAdmin
}

// repoLookupTopLevel represents a top-level repo (repo)
type repoLookupTopLevel struct {
	Name string
}

func (rl repoLookupTopLevel) Path() string {
	return path.Join("top-level", rl.Name)
}

func (rl repoLookupTopLevel) IsValid(c *AdminConfig) bool {
	// A top level repo is valid if the repo is defined.
	_, ok := c.Repos[rl.Name]
	return ok
}

func (rl repoLookupTopLevel) UserHasAccess(c *AdminConfig, u *User, a AccessType) bool {
	return genericUserHasAccess(c.Repos[rl.Name], u, a)
}

// repoLookupUserConfig represents a user config repo (~user)
type repoLookupUserConfig struct {
	User string
}

func (rl repoLookupUserConfig) Path() string {
	return path.Join("admin", "user-"+rl.User)
}

func (rl repoLookupUserConfig) IsValid(c *AdminConfig) bool {
	// A user config repo is valid if the user exists and is not disabled.
	user, ok := c.Users[rl.User]
	if !ok {
		return false
	}

	return !user.Disabled
}

func (rl repoLookupUserConfig) UserHasAccess(c *AdminConfig, u *User, a AccessType) bool {
	if u.IsAdmin {
		return true
	}

	return u.Username == rl.User
}

// repoLookupUser represents a user repo (~user/repo)
type repoLookupUser struct {
	User string
	Name string
}

func (rl repoLookupUser) Path() string {
	return path.Join("users", rl.User, rl.Name)
}

func (rl repoLookupUser) IsValid(c *AdminConfig) bool {
	// A user repo is valid if the user exists and the repo is defined.
	user, ok := c.Users[rl.User]
	if !ok {
		return false
	}

	_, ok = user.Repos[rl.Name]
	return ok
}

func (rl repoLookupUser) UserHasAccess(c *AdminConfig, u *User, a AccessType) bool {
	if u.IsAdmin {
		return true
	}

	if u.Username == rl.User {
		return true
	}

	return genericUserHasAccess(c.Users[rl.User].Repos[rl.Name], u, a)
}

// repoLookupOrgConfig represents an org config repo (@org)
type repoLookupOrgConfig struct {
	Org string
}

func (rl repoLookupOrgConfig) Path() string {
	return path.Join("admin", "org-"+rl.Org)
}

func (rl repoLookupOrgConfig) IsValid(c *AdminConfig) bool {
	// A org config repo is valid if the org exists.
	_, ok := c.Orgs[rl.Org]
	return ok
}

func (rl repoLookupOrgConfig) UserHasAccess(c *AdminConfig, u *User, a AccessType) bool {
	if u.IsAdmin {
		return true
	}

	return listContains(c.Orgs[rl.Org].Admin, u.Username)
}

// repoLookupOrg represents an org repo (@org/repo)
type repoLookupOrg struct {
	Org  string
	Name string
}

func (rl repoLookupOrg) Path() string {
	return path.Join("orgs", rl.Org, rl.Name)
}

func (rl repoLookupOrg) IsValid(c *AdminConfig) bool {
	// A org repo is valid if the org exists and the repo is defined.
	org, ok := c.Orgs[rl.Org]
	if !ok {
		return false
	}

	_, ok = org.Repos[rl.Name]
	return ok
}

func (rl repoLookupOrg) UserHasAccess(c *AdminConfig, u *User, a AccessType) bool {
	if u.IsAdmin {
		return true
	}

	org := c.Orgs[rl.Org]

	// If an org write user is requesting write or below, they can access the repo.
	if listContains(org.Write, u.Username) && a <= AccessTypeWrite {
		return true
	}

	// If a org read user is requesting read or below, they can access the repo.
	if listContains(org.Read, u.Username) && a <= AccessTypeRead {
		return true
	}

	// Fall back to generic repo permission check.
	return genericUserHasAccess(org.Repos[rl.Name], u, a)
}
