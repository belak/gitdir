package main

import (
	"errors"
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

// GetUserFromKey looks up a user object given their PublicKey.
func (ac *AdminConfig) GetUserFromKey(pk PublicKey) (*User, error) {
	usernames, ok := ac.PublicKeys[pk.RawMarshalAuthorizedKey()]
	if !ok {
		return AnonymousUser, errors.New("key does not match a user")
	}

	if len(usernames) != 1 {
		return AnonymousUser, errors.New("key matches multiple users")
	}

	userConfig, ok := ac.Users[usernames[0]]
	if !ok {
		return AnonymousUser, errors.New("key does not match a user")
	}

	return &User{
		Username:    usernames[0],
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}

// GetUserFromInvite looks up a user object given an invite code.
func (ac *AdminConfig) GetUserFromInvite(invite string) (*User, error) {
	username, ok := ac.Invites[invite]
	if !ok {
		return AnonymousUser, errors.New("invite does not match a user")
	}

	userConfig, ok := ac.Users[username]
	if !ok {
		return AnonymousUser, errors.New("invite does not match a user")
	}

	return &User{
		Username:    username,
		IsAnonymous: false,
		IsAdmin:     userConfig.IsAdmin,
	}, nil
}
