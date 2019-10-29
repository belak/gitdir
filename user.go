package main

import (
	"errors"
)

type User struct {
	Username    string
	IsAnonymous bool
	IsAdmin     bool
}

var AnonymousUser = &User{
	Username:    "<anonymous>",
	IsAnonymous: true,
}

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
