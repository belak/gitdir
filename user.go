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

// LookupUserFromKey looks up a user object given their PublicKey.
func (serv *Server) LookupUserFromKey(pk models.PublicKey, remoteUser string) (*User, error) {
	serv.lock.RLock()
	defer serv.lock.RUnlock()

	username, ok := serv.publicKeys[pk.RawMarshalAuthorizedKey()]
	if !ok {
		log.Warn().Msg("key does not exist")
		return AnonymousUser, ErrUserNotFound
	}

	userConfig, ok := serv.settings.Users[username]
	if !ok {
		log.Warn().Msg("key does not match a user")
		return AnonymousUser, ErrUserNotFound
	}

	// If they weren't the git user make sure their username matches their key.
	if remoteUser != serv.settings.Options.GitUser && remoteUser != username {
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
func (serv *Server) LookupUserFromInvite(invite string) (*User, error) {
	serv.lock.RLock()
	defer serv.lock.RUnlock()

	username, ok := serv.settings.Invites[invite]
	if !ok {
		log.Warn().Msg("invite does not exist")
		return AnonymousUser, ErrUserNotFound
	}

	userConfig, ok := serv.settings.Users[username]
	if !ok {
		log.Warn().Msg("invite does not match a user")
		return AnonymousUser, ErrUserNotFound
	}

	return &User{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}
