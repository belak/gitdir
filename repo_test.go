package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var repoTestCases = []struct {
	In     string
	Type   RepoLookup
	Path   string
	Errors bool
}{
	{
		In:   "admin",
		Type: &repoLookupAdmin{},
		Path: "admin/admin",
	},
	{
		In:   "top-level",
		Type: &repoLookupTopLevel{},
		Path: "top-level/top-level",
	},
	{
		In:   "@org",
		Type: &repoLookupOrgConfig{},
		Path: "admin/org-org",
	},
	{
		In:   "@org/repo",
		Type: &repoLookupOrg{},
		Path: "orgs/org/repo",
	},
	{
		In:   "~user",
		Type: &repoLookupUserConfig{},
		Path: "admin/user-user",
	},
	{
		In:   "~user/repo",
		Type: &repoLookupUser{},
		Path: "users/user/repo",
	},
}

func TestParseRepo(t *testing.T) {
	for _, testCase := range repoTestCases {
		lookup, err := ParseRepo(&defaultAdminOptions, testCase.In)

		if testCase.Errors {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)
		assert.IsType(t, testCase.Type, lookup)
		assert.Equal(t, testCase.Path, lookup.Path())
	}
}
