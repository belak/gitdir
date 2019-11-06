package gitdir

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// AccessType represents the level of access being requested and the level of
// access a user has.
type AccessType int

// AccessType defaults to AccessTypeNone for security. A repo lookup returns the
// level of permissions a user has and if it's not explicitly set, they don't
// have any.
const (
	AccessTypeNone AccessType = iota
	AccessTypeRead
	AccessTypeWrite
	AccessTypeAdmin
)

// String implements Stringer
func (a AccessType) String() string {
	switch a {
	case AccessTypeNone:
		return "None"
	case AccessTypeRead:
		return "Read"
	case AccessTypeWrite:
		return "Write"
	case AccessTypeAdmin:
		return "Admin"
	default:
		return fmt.Sprintf("Unknown(%d)", a)
	}
}

const groupPrefix = "$"

func (c *Config) doesGroupContainUser(username string, groupName string, groupPath []string) bool {
	// Group loop - this should never be possible in a checked config.
	if listContainsStr(groupPath, groupName) {
		log.Warn().Strs("groups", append(groupPath, groupName)).Msg("group loop")
		return false
	}

	groupPath = append(groupPath, groupName)

	for _, lookup := range c.Groups[groupName] {
		if strings.HasPrefix(lookup, groupPrefix) {
			intGroupName := strings.TrimPrefix(lookup, groupPrefix)

			if c.doesGroupContainUser(username, intGroupName, groupPath) {
				return true
			}
		}

		if lookup == username {
			return true
		}
	}

	return false
}

func (c *Config) checkListsForUser(username string, userLists ...[]string) bool {
	for _, list := range userLists {
		for _, lookup := range list {
			if strings.HasPrefix(lookup, groupPrefix) {
				if c.doesGroupContainUser(username, strings.TrimPrefix(lookup, groupPrefix), nil) {
					return true
				}
			} else {
				if lookup == username {
					return true
				}
			}
		}
	}

	return false
}

// TODO: clean up nolint here
func (c *Config) checkUserRepoAccess(user *User, repo *RepoLookup) AccessType { //nolint:funlen
	// Admins always have access to everything.
	if user.IsAdmin {
		return AccessTypeAdmin
	}

	switch repo.Type {
	case RepoTypeAdmin:
		// If we made it this far, they're not an admin, so they don't have
		// access.
		return AccessTypeNone
	case RepoTypeOrgConfig:
		org := c.Orgs[repo.PathParts[0]]
		if c.checkListsForUser(user.Username, org.Admin) {
			return AccessTypeAdmin
		}

		return AccessTypeNone
	case RepoTypeOrg:
		org := c.Orgs[repo.PathParts[0]]

		// Because we already checked to see if this repo exists, this user has
		// admin on the repo if they're an org admin.
		if c.checkListsForUser(user.Username, org.Admin) {
			return AccessTypeAdmin
		}

		repo := org.Repos[repo.PathParts[1]]
		if repo == nil {
			// If this is an implicitly created repo, we can only check the org
			// level permissions.
			if c.Options.ImplicitRepos {
				switch {
				case c.checkListsForUser(user.Username, org.Write):
					return AccessTypeWrite
				case c.checkListsForUser(user.Username, org.Read):
					return AccessTypeRead
				}
			}

			return AccessTypeNone
		}

		switch {
		case c.checkListsForUser(user.Username, org.Write, repo.Write):
			return AccessTypeWrite
		case c.checkListsForUser(user.Username, org.Read, repo.Read):
			return AccessTypeRead
		}

		return AccessTypeNone
	case RepoTypeUserConfig:
		if repo.PathParts[0] == user.Username {
			return AccessTypeAdmin
		}

		return AccessTypeNone
	case RepoTypeUser:
		// Because we already checked to see if this repo exists, the user has
		// admin on the repo if they own the repo.
		if repo.PathParts[0] == user.Username {
			return AccessTypeAdmin
		}

		userConfig := c.Users[repo.PathParts[0]]
		repo := userConfig.Repos[repo.PathParts[1]]

		// Only the given user has access to implicit repos, so if the repo
		// isn't explicitly defined, noone else has access.
		if repo == nil {
			return AccessTypeNone
		}

		switch {
		case c.checkListsForUser(user.Username, repo.Write):
			return AccessTypeWrite
		case c.checkListsForUser(user.Username, repo.Read):
			return AccessTypeRead
		}
	case RepoTypeTopLevel:
		repo := c.Repos[repo.PathParts[0]]
		if repo == nil {
			// Only admins have access to implicitly created top-level repos.
			return AccessTypeNone
		}

		switch {
		case c.checkListsForUser(user.Username, repo.Write):
			return AccessTypeWrite
		case c.checkListsForUser(user.Username, repo.Read):
			return AccessTypeRead
		}
	}

	return AccessTypeNone
}
