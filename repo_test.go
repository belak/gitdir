package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var repoTestCases = []struct {
	In     string
	Type   RepoType
	Path   string
	Errors bool
}{
	{
		In:   "admin",
		Type: RepoTypeAdmin,
		Path: "admin/admin",
	},
	{
		In:   "@org",
		Type: RepoTypeOrgConfig,
		Path: "admin/org-org",
	},
	{
		In:   "@org/repo",
		Type: RepoTypeOrg,
		Path: "orgs/org/repo",
	},
	{
		In:   "~user",
		Type: RepoTypeUserConfig,
		Path: "admin/user-user",
	},
	{
		In:   "~user/repo",
		Type: RepoTypeUser,
		Path: "users/user/repo",
	},
}

func TestParseRepo(t *testing.T) {
	c := NewDefaultConfig()
	c.BasePath = "."
	for _, testCase := range repoTestCases {
		lookup, err := ParseRepo(c, testCase.In)
		if testCase.Errors {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)

		assert.Equal(t, testCase.Type, lookup.Type)
		assert.Equal(t, testCase.Path, lookup.Path)
	}
}
