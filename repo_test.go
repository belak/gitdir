package gitdir

import (
	"testing"

	"github.com/belak/go-gitdir/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

func mustParsePK(data string) models.PublicKey {
	pk, err := models.ParsePublicKey([]byte(data))
	if err != nil {
		panic(err.Error())
	}

	return *pk
}

func newTestRepoConfig() *models.RepoConfig {
	repo := models.NewRepoConfig()
	repo.Write = []string{"write-user"}
	repo.Read = []string{"read-user"}

	return repo
}

func newTestConfig() *Config { //nolint:funlen
	c := NewConfig(memfs.New())

	// Define some basic users
	c.Users["an-admin"] = models.NewAdminConfigUser()
	c.Users["an-admin"].IsAdmin = true
	c.Users["an-admin"].Keys = []models.PublicKey{
		mustParsePK("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILQGpcX2owFW6hdTWHa/CzbTwhUJlmI8gKAgnp/c0NK2 an-admin"),
	}

	c.Users["non-admin"] = models.NewAdminConfigUser()
	c.Users["non-admin"].Repos["test-repo"] = newTestRepoConfig()

	// Org-level permissions
	c.Users["org-admin"] = models.NewAdminConfigUser()
	c.Users["org-write"] = models.NewAdminConfigUser()
	c.Users["org-read"] = models.NewAdminConfigUser()

	// Repo-level permissions
	c.Users["read-user"] = models.NewAdminConfigUser()
	c.Users["write-user"] = models.NewAdminConfigUser()
	c.Users["nothing-user"] = models.NewAdminConfigUser()

	c.Users["disabled"] = models.NewAdminConfigUser()
	c.Users["disabled"].Disabled = true
	c.Users["disabled"].Keys = []models.PublicKey{
		mustParsePK("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBx4DYr9m+EnG0tgFsUIZqrDP7pa+vpVXJJ6/PE9J7Ll disabled"),
	}

	// Basic group
	c.Groups["admins"] = []string{"an-admin"}
	c.Groups["nested-admins"] = []string{"$admins"}

	// Group loop
	c.Groups["loop"] = []string{"$loop1"}
	c.Groups["loop1"] = []string{"$loop2"}
	c.Groups["loop2"] = []string{"$loop1"}

	// Basic org
	c.Orgs["an-org"] = models.NewOrgConfig()
	c.Orgs["an-org"].Admin = []string{"org-admin"}
	c.Orgs["an-org"].Write = []string{"org-write"}
	c.Orgs["an-org"].Read = []string{"org-read"}
	c.Orgs["an-org"].Repos["test-repo"] = newTestRepoConfig()

	c.Repos["test-repo"] = newTestRepoConfig()

	c.Invites["valid-invite"] = "an-admin"
	c.Invites["user-missing"] = "invalid-user"
	c.Invites["user-disabled"] = "disabled"

	// Force all settings repos to "on"
	c.Options.UserConfigRepos = true
	c.Options.OrgConfig = true
	c.Options.OrgConfigRepos = true

	c.flatten()

	// Insert a bogus PK which points to a user that doesn't exist.
	c.publicKeys["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQJzT5mM5eDYhoe3pVodWPCDzoj0/+pCVNoVsuUR4ao"] = "invalid-user"

	return c
}

func TestRepoTypeStringer(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Admin", RepoTypeAdmin.String())
	assert.Equal(t, "OrgConfig", RepoTypeOrgConfig.String())
	assert.Equal(t, "Org", RepoTypeOrg.String())
	assert.Equal(t, "UserConfig", RepoTypeUserConfig.String())
	assert.Equal(t, "User", RepoTypeUser.String())
	assert.Equal(t, "TopLevel", RepoTypeTopLevel.String())
	assert.Equal(t, "Unknown(42)", RepoType(42).String())
}

func TestAccessTypeStringer(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "None", AccessTypeNone.String())
	assert.Equal(t, "Read", AccessTypeRead.String())
	assert.Equal(t, "Write", AccessTypeWrite.String())
	assert.Equal(t, "Admin", AccessTypeAdmin.String())
	assert.Equal(t, "Unknown(42)", AccessType(42).String())
}

func TestRepoLookup(t *testing.T) { //nolint:funlen
	t.Parallel()

	c := newTestConfig()

	var tests = []struct {
		Lookup string
		Type   RepoType
		Path   string
		Err    error
	}{
		{
			"admin",
			RepoTypeAdmin,
			"admin/admin",
			nil,
		},
		{
			"@an-org",
			RepoTypeOrgConfig,
			"admin/org-an-org",
			nil,
		},
		{
			"@an-org/test-repo",
			RepoTypeOrg,
			"orgs/an-org/test-repo",
			nil,
		},
		{
			"~non-admin",
			RepoTypeUserConfig,
			"admin/user-non-admin",
			nil,
		},
		{
			"~non-admin/test-repo",
			RepoTypeUser,
			"users/non-admin/test-repo",
			nil,
		},
		{
			"test-repo",
			RepoTypeTopLevel,
			"top-level/test-repo",
			nil,
		},

		{
			"@an-org/repo/invalid",
			RepoTypeOrg,
			"",
			ErrInvalidRepoFormat,
		},
		{
			"@invalid-org",
			RepoTypeOrgConfig,
			"",
			ErrRepoDoesNotExist,
		},
		{
			"@an-org/does-not-exist",
			RepoTypeOrg,
			"",
			ErrRepoDoesNotExist,
		},

		{
			"~non-admin/repo/invalid",
			RepoTypeUser,
			"",
			ErrInvalidRepoFormat,
		},
		{
			"~invalid-user",
			RepoTypeOrgConfig,
			"",
			ErrRepoDoesNotExist,
		},
		{
			"~non-admin/does-not-exist",
			RepoTypeOrg,
			"",
			ErrRepoDoesNotExist,
		},

		{
			"top-level/invalid",
			RepoTypeTopLevel,
			"",
			ErrInvalidRepoFormat,
		},
		{
			"invalid-repo",
			RepoTypeTopLevel,
			"",
			ErrRepoDoesNotExist,
		},
	}

	for _, test := range tests {
		lookup, err := c.lookupRepo(test.Lookup)

		if test.Err != nil {
			assert.Equal(t, test.Err, err)
			continue
		}

		require.Nil(t, err)

		assert.Equal(t, test.Type, lookup.Type)
		assert.Equal(t, test.Path, lookup.Path())
	}

	invalidLookup := &RepoLookup{
		Type: RepoType(42),
	}
	assert.Equal(t, "/dev/null", invalidLookup.Path())
}

func TestDoesGroupContainUser(t *testing.T) {
	t.Parallel()

	c := newTestConfig()

	assert.True(t, c.doesGroupContainUser("an-admin", "admins", nil))
	assert.True(t, c.doesGroupContainUser("an-admin", "nested-admins", nil))
	assert.False(t, c.doesGroupContainUser("non-admin", "admins", nil))
	assert.False(t, c.doesGroupContainUser("non-admin", "nested-admins", nil))
	assert.False(t, c.doesGroupContainUser("an-admin", "loop", nil))
}

func TestCheckListsForUser(t *testing.T) {
	t.Parallel()

	c := newTestConfig()

	// Basic checks
	assert.False(t, c.checkListsForUser("an-admin"))
	assert.True(t, c.checkListsForUser("an-admin", []string{"an-admin"}))
	assert.True(t, c.checkListsForUser("an-admin", []string{"$admins"}))
	assert.True(t, c.checkListsForUser("an-admin", []string{"$nested-admins"}))

	// Ensure loops don't crash this
	assert.False(t, c.checkListsForUser("an-admin", []string{"$loop"}))
}

type allRepoAccessLevels struct {
	Admin      AccessType
	OrgConfig  AccessType
	OrgRepo    AccessType
	UserConfig AccessType
	UserRepo   AccessType
	TopLevel   AccessType
}

type allImplicitAccessLevels struct {
	Org      AccessType
	User     AccessType
	TopLevel AccessType
}

func lookupAndCheck(t *testing.T, c *Config, u *User, path string, access AccessType) {
	repo, err := c.LookupRepoAccess(u, path)
	require.Nil(t, err)
	require.NotNil(t, repo)
	assert.Equal(t, access, repo.Access)
}

func testCheckRepoAccess(t *testing.T, c *Config, u *User, access allRepoAccessLevels) {
	lookupAndCheck(t, c, u, "admin", access.Admin)
	lookupAndCheck(t, c, u, "@an-org", access.OrgConfig)
	lookupAndCheck(t, c, u, "@an-org/test-repo", access.OrgRepo)
	lookupAndCheck(t, c, u, "~non-admin", access.UserConfig)
	lookupAndCheck(t, c, u, "~non-admin/test-repo", access.UserRepo)
	lookupAndCheck(t, c, u, "test-repo", access.TopLevel)
}

func testImplicitRepoAccess(t *testing.T, c *Config, u *User, access allImplicitAccessLevels) {
	prevImplicit := c.Options.ImplicitRepos

	defer func() {
		c.Options.ImplicitRepos = prevImplicit
	}()

	c.Options.ImplicitRepos = true

	lookupAndCheck(t, c, u, "@an-org/implicit", access.Org)
	lookupAndCheck(t, c, u, "~non-admin/implicit", access.User)
	lookupAndCheck(t, c, u, "implicit", access.TopLevel)
}

func TestCheckUserRepoAccess(t *testing.T) { //nolint:funlen
	t.Parallel()

	c := newTestConfig()

	// Permission checking access level table
	//
	// |----------------+-------+------------+----------+-------------+-----------+-----------|
	// |                | Admin | Org Config | Org Repo | User Config | User Repo | Top Level |
	// |----------------+-------+------------+----------+-------------+-----------+-----------|
	// | Admin          | Admin | Admin      | Admin    | Admin       | Admin     | Admin     |
	// | Org Admin      |       | Admin      | Admin    |             |           |           |
	// | Org Writer     |       |            | Write    |             |           |           |
	// | Org Reader     |       |            | Read     |             |           |           |
	// | Non-Admin User |       |            |          | Admin       | Admin     |           |           |
	// | Direct Writer  |       |            |          |             |           | Write     |
	// | Direct Reader  |       |            |          |             |           | Read      |
	// |----------------+-------+------------+----------+-------------+-----------+-----------|
	//
	// Implicit repos add the following:
	//
	// |---------------+-----------+-------+-------|
	// |               | Top Level | Org   | User  |
	// |---------------+-----------+-------+-------|
	// | Admin         | Admin     | Admin | Admin |
	// | Org Admin     |           | Admin |       |
	// | Org Writer    |           | Write |       |
	// | Org Reader    |           | Read  |       |
	// | User          |           |       | Admin |
	// | Direct Writer |           |       |       |
	// | Direct Reader |           |       |       |
	// |---------------+-----------+-------+-------|

	var tests = []struct {
		Username       string
		Access         allRepoAccessLevels
		ImplicitAccess allImplicitAccessLevels
	}{
		{
			"an-admin",
			allRepoAccessLevels{
				Admin:      AccessTypeAdmin,
				OrgConfig:  AccessTypeAdmin,
				OrgRepo:    AccessTypeAdmin,
				UserConfig: AccessTypeAdmin,
				UserRepo:   AccessTypeAdmin,
				TopLevel:   AccessTypeAdmin,
			},
			allImplicitAccessLevels{
				Org:      AccessTypeAdmin,
				User:     AccessTypeAdmin,
				TopLevel: AccessTypeAdmin,
			},
		},
		{
			"org-admin",
			allRepoAccessLevels{
				OrgConfig: AccessTypeAdmin,
				OrgRepo:   AccessTypeAdmin,
			},
			allImplicitAccessLevels{
				Org: AccessTypeAdmin,
			},
		},
		{
			"org-write",
			allRepoAccessLevels{
				OrgRepo: AccessTypeWrite,
			},
			allImplicitAccessLevels{
				Org: AccessTypeWrite,
			},
		},
		{
			"org-read",
			allRepoAccessLevels{
				OrgRepo: AccessTypeRead,
			},
			allImplicitAccessLevels{
				Org: AccessTypeRead,
			},
		},
		{
			"non-admin",
			allRepoAccessLevels{
				UserConfig: AccessTypeAdmin,
				UserRepo:   AccessTypeAdmin,
			},
			allImplicitAccessLevels{
				User: AccessTypeAdmin,
			},
		},
		{
			"write-user",
			allRepoAccessLevels{
				OrgRepo:  AccessTypeWrite,
				UserRepo: AccessTypeWrite,
				TopLevel: AccessTypeWrite,
			},
			allImplicitAccessLevels{},
		},
		{
			"read-user",
			allRepoAccessLevels{
				OrgRepo:  AccessTypeRead,
				UserRepo: AccessTypeRead,
				TopLevel: AccessTypeRead,
			},
			allImplicitAccessLevels{},
		},
		{
			"nothing-user",
			allRepoAccessLevels{},
			allImplicitAccessLevels{},
		},
	}

	for _, test := range tests {
		user, err := c.LookupUserFromUsername(test.Username)
		require.Nil(t, err)

		testCheckRepoAccess(t, c, user, test.Access)
		testImplicitRepoAccess(t, c, user, test.ImplicitAccess)
	}

	// One special test case to check a repo that doesn't exist.
	repo, err := c.LookupRepoAccess(&User{Username: "an-admin", IsAdmin: true}, "invalid-repo")
	require.Equal(t, ErrRepoDoesNotExist, err)
	require.Nil(t, repo)
}
