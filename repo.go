package main

import (
	"errors"
	"strings"
)

type AccessType int

const (
	AccessTypeRead AccessType = iota
	AccessTypeWrite
	AccessTypeAdmin
)

var ErrInvalidRepoFormat = errors.New("invalid repo format")

func ParseRepo(c *Config, pathname string) (RepoLookup, error) {
	if pathname == "admin" {
		return &repoLookupAdmin{}, nil
	}

	if strings.HasPrefix(pathname, c.OrgPrefix) {
		// Strip off the org prefix and continue parsing
		pathname = pathname[len(c.OrgPrefix):]

		path := strings.Split(pathname, "/")
		if len(path) == 1 {
			return &repoLookupOrgConfig{
				Org: path[0],
			}, nil
		}

		if len(path) == 2 {
			return &repoLookupOrg{
				Org:  path[0],
				Name: path[1],
			}, nil
		}

		return nil, ErrInvalidRepoFormat
	}

	if strings.HasPrefix(pathname, c.UserPrefix) {
		// Strip off the org prefix and continue parsing
		pathname = pathname[len(c.UserPrefix):]

		path := strings.Split(pathname, "/")
		if len(path) == 1 {
			return &repoLookupUserConfig{
				User: path[0],
			}, nil
		}

		if len(path) == 2 {
			return &repoLookupUser{
				User: path[0],
				Name: path[1],
			}, nil
		}

		return nil, ErrInvalidRepoFormat
	}

	path := strings.Split(pathname, "/")
	if len(path) == 1 {
		return &repoLookupTopLevel{
			Name: path[0],
		}, nil
	}

	return nil, ErrInvalidRepoFormat
}
