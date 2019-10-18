package main

const (
	accessTypeUnknown accessType = iota
	accessTypeRead
	accessTypeWrite
)

func (c *adminConfig) UserHasRepoAccess(user *User, repo *RepoLookup, access accessType) bool {
	// This shouldn't be possible
	if user.IsAnonymous {
		return false
	}

	// Admins are super-admins. They always have access to all repos.
	if user.IsAdmin {
		return true
	}

	// Top level repos need to have permissions explicitly defined
	if repo.Type == RepoTypeTopLevel {
		r, ok := c.Repos[repo.Name]
		if !ok {
			return false
		}

		// If a write user is requesting write or below, they can access the repo
		if listContains(r.Write, user.Username) && access <= accessTypeWrite {
			return true
		}

		// If a read user is requesting read or below, they can access the repo
		if listContains(r.Read, user.Username) && access <= accessTypeRead {
			return true
		}

		return false
	}

	// Users have all access levels in their own user dir
	if repo.Type == RepoTypeUser {
		// If user repos aren't enabled, we need to bail
		if !c.Options.UserRepos {
			return false
		}

		// If the user is accessing their own repos, they have access.
		if user.Username == repo.Dir {
			return true
		}

		// Otherwise, they don't have access
		return false
	}

	if repo.Type == RepoTypeUserConfig {
		// If the user is accessing their own repos, they have access.
		return user.Username == repo.Name
	}

	if repo.Type == RepoTypeOrg {
		// If the org doesn't exist, no one has access
		org, ok := c.Orgs[repo.Dir]
		if !ok {
			return false
		}

		// Org admins can do whatever they want in their org
		if listContains(org.Admin, user.Username) {
			return true
		}

		// If a write user is requesting write or below, they can access the repo
		if listContains(org.Write, user.Username) && access <= accessTypeWrite {
			return true
		}

		// If a read user is requesting read or below, they can access the repo
		if listContains(org.Read, user.Username) && access <= accessTypeRead {
			return true
		}

		return false
	}

	if repo.Type == RepoTypeOrgConfig {
		// If the org doesn't exist, no one has access
		org, ok := c.Orgs[repo.Name]
		if !ok {
			return false
		}

		// Org admins can do whatever they want in their org
		if listContains(org.Admin, user.Username) {
			return true
		}

		return false
	}

	return false
}
