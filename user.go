package gitdir

import (
	"errors"

	"github.com/belak/go-gitdir/models"
	"github.com/rs/zerolog/log"
)

// UserSession is the internal representation of a user. This data is copied
// from the loaded config file and tagged with the PublicKey used in this
// session.
type UserSession struct {
	Username    string
	IsAnonymous bool
	IsAdmin     bool

	PublicKey models.PublicKey
}

var AnonymousUserSession = &UserSession{
	Username:    "<anonymous>",
	IsAnonymous: true,
}

// ErrUserNotFound is returned from LookupUser commands when
var ErrUserNotFound = errors.New("user not found")

// LookupUserFromKey looks up a user object given their PublicKey.
func (c *Config) LookupUserFromPublicKey(pk models.PublicKey, usernameHint string) (*UserSession, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	username, ok := c.publicKeys[pk.RawMarshalAuthorizedKey()]
	if !ok {
		log.Warn().Msg("key does not exist")
		return nil, ErrUserNotFound
	}

	userConfig, ok := c.adminConfig.Users[username]
	if !ok {
		log.Warn().Msg("key does not match a user")
		return nil, ErrUserNotFound
	}

	if userConfig.Disabled {
		log.Warn().Msg("user is disabled")
		return nil, ErrUserNotFound
	}

	// If they weren't the git user make sure their username matches their key.
	if usernameHint != c.adminConfig.Options.GitUser && usernameHint != username {
		log.Warn().Msg("key belongs to different user")
		return nil, ErrUserNotFound
	}

	return &UserSession{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,

		PublicKey: pk,
	}, nil
}

// LookupUserFromInvite looks up a user object given an invite code.
func (c *Config) LookupUserFromInvite(invite string) (*UserSession, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	username, ok := c.adminConfig.Invites[invite]
	if !ok {
		log.Warn().Msg("invite does not exist")
		return nil, ErrUserNotFound
	}

	userConfig, ok := c.adminConfig.Users[username]
	if !ok {
		log.Warn().Msg("invite does not match a user")
		return nil, ErrUserNotFound
	}

	if userConfig.Disabled {
		log.Warn().Msg("user is disabled")
		return nil, ErrUserNotFound
	}

	return &UserSession{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}
