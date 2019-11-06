package gitdir

import (
	"errors"

	"github.com/belak/go-gitdir/models"
	"github.com/rs/zerolog/log"
)

// User is the internal representation of a user. This data is copied from the
// loaded config file.
type User struct {
	Username    string
	IsAnonymous bool
	IsAdmin     bool
}

// AnonymousUser is the user that is returned when no user is available.
var AnonymousUser = &User{
	Username:    "<anonymous>",
	IsAnonymous: true,
}

// ErrUserNotFound is returned from LookupUser commands when
var ErrUserNotFound = errors.New("user not found")

// LookupUserFromUsername looks up a user objects given their username.
func (c *Config) LookupUserFromUsername(username string) (*User, error) {
	userConfig, ok := c.Users[username]
	if !ok {
		log.Warn().Msg("username does not match a user")
		return AnonymousUser, ErrUserNotFound
	}

	if userConfig.Disabled {
		log.Warn().Msg("user is disabled")
		return AnonymousUser, ErrUserNotFound
	}

	return &User{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}

// LookupUserFromKey looks up a user object given their PublicKey.
func (c *Config) LookupUserFromKey(pk models.PublicKey, remoteUser string) (*User, error) {
	username, ok := c.publicKeys[pk.RawMarshalAuthorizedKey()]
	if !ok {
		log.Warn().Msg("key does not exist")
		return AnonymousUser, ErrUserNotFound
	}

	userConfig, ok := c.Users[username]
	if !ok {
		log.Warn().Msg("key does not match a user")
		return AnonymousUser, ErrUserNotFound
	}

	if userConfig.Disabled {
		log.Warn().Msg("user is disabled")
		return AnonymousUser, ErrUserNotFound
	}

	// If they weren't the git user make sure their username matches their key.
	if remoteUser != c.Options.GitUser && remoteUser != username {
		log.Warn().Msg("key belongs to different user")
		return AnonymousUser, ErrUserNotFound
	}

	return &User{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}

// LookupUserFromInvite looks up a user object given an invite code.
func (c *Config) LookupUserFromInvite(invite string) (*User, error) {
	username, ok := c.Invites[invite]
	if !ok {
		log.Warn().Msg("invite does not exist")
		return AnonymousUser, ErrUserNotFound
	}

	userConfig, ok := c.Users[username]
	if !ok {
		log.Warn().Msg("invite does not match a user")
		return AnonymousUser, ErrUserNotFound
	}

	if userConfig.Disabled {
		log.Warn().Msg("user is disabled")
		return AnonymousUser, ErrUserNotFound
	}

	return &User{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}
