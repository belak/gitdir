package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupMembers(t *testing.T) {
	goodGroups := map[string][]string{
		"me": {
			"belak",
		},
		"admins": {
			"$me",
		},
	}

	nestedGroups := map[string][]string{
		"me": {
			"belak",
		},
		"more-admins": {
			"belak",
			"another-user",
		},
		"admins": {
			"$me",
			"another-user",
			"$more-admins",
		},
	}

	badGroups := map[string][]string{
		"me":     {"$admins"},
		"admins": {"$me"},
	}

	users, err := groupMembers(goodGroups, "admins", nil)
	assert.NoError(t, err)
	assert.ElementsMatch(t, users, []string{"belak"})

	users, err = groupMembers(nestedGroups, "admins", nil)
	assert.NoError(t, err)
	assert.ElementsMatch(t, users, []string{"another-user", "belak"})

	users, err = groupMembers(badGroups, "admins", nil)
	assert.Error(t, err)
	assert.Empty(t, users)
}
