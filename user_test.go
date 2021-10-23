package gitdir

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/belak/go-gitdir/models"
)

func TestLookupUserFromUsername(t *testing.T) {
	t.Parallel()

	c := newTestConfig()

	var tests = []struct { //nolint:gofumpt
		Username string
		Error    error
	}{
		{
			"missing-user",
			ErrUserNotFound,
		},
		{
			"disabled",
			ErrUserNotFound,
		},
	}

	for _, test := range tests {
		user, err := c.LookupUserFromUsername(test.Username)

		if test.Error != nil {
			require.Equal(t, test.Error, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, test.Username, user.Username)
		}
	}
}

func TestLookupUserFromKey(t *testing.T) { //nolint:funlen
	t.Parallel()

	c := newTestConfig()

	var tests = []struct { //nolint:gofumpt
		UserHint     string
		Username     string
		PublicKey    string
		Error        error
		ErrorGitUser error
	}{
		{
			"missing-user",
			"missing-user",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJ7+BNW+C5HHQ8C3QcJCYfUvxz+biXbxB0JtufT+P2AD user-not-found",
			ErrUserNotFound,
			ErrUserNotFound,
		},
		{
			"disabled",
			"disabled",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBx4DYr9m+EnG0tgFsUIZqrDP7pa+vpVXJJ6/PE9J7Ll disabled",
			ErrUserNotFound,
			ErrUserNotFound,
		},
		{
			"invalid-user",
			"invalid-user",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQJzT5mM5eDYhoe3pVodWPCDzoj0/+pCVNoVsuUR4ao invalid-user",
			ErrUserNotFound,
			ErrUserNotFound,
		},

		{
			"an-admin",
			"an-admin",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILQGpcX2owFW6hdTWHa/CzbTwhUJlmI8gKAgnp/c0NK2 an-admin",
			nil,
			nil,
		},
		{
			// Mismatched username will error
			"non-admin",
			"an-admin",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILQGpcX2owFW6hdTWHa/CzbTwhUJlmI8gKAgnp/c0NK2 an-admin",
			ErrUserNotFound,
			nil,
		},
	}

	for _, test := range tests {
		pk, err := models.ParsePublicKey([]byte(test.PublicKey))
		require.Nil(t, err)

		// Try with the username hint
		user, err := c.LookupUserFromKey(*pk, test.UserHint)

		if test.Error != nil {
			require.Equal(t, test.Error, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, test.Username, user.Username)
		}

		// Try without the username hint
		user, err = c.LookupUserFromKey(*pk, c.Options.GitUser)

		if test.ErrorGitUser != nil {
			require.Equal(t, test.ErrorGitUser, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, test.Username, user.Username)
		}
	}
}

func TestLookupUserFromInvite(t *testing.T) {
	t.Parallel()

	c := newTestConfig()

	var tests = []struct { //nolint:gofumpt
		Username string
		Invite   string
		Error    error
	}{
		{
			"an-admin",
			"valid-invite",
			nil,
		},
		{
			"an-admin",
			"invalid-invite",
			ErrUserNotFound,
		},
		{
			"disabled",
			"user-disabled",
			ErrUserNotFound,
		},
		{
			"invalid-user",
			"user-missing",
			ErrUserNotFound,
		},
	}

	for _, test := range tests {
		user, err := c.LookupUserFromInvite(test.Invite)

		if test.Error != nil {
			require.Equal(t, test.Error, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, test.Username, user.Username)
		}
	}
}
