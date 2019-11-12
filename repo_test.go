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
	c.adminConfig.Users["an-admin"] = models.NewAdminConfigUser()
	c.adminConfig.Users["an-admin"].IsAdmin = true
	c.adminConfig.Users["an-admin"].Keys = []models.PublicKey{
		mustParsePK("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILQGpcX2owFW6hdTWHa/CzbTwhUJlmI8gKAgnp/c0NK2 an-admin"),
	}
	c.users["an-admin"] = models.NewUserConfig()

	c.adminConfig.Users["non-admin"] = models.NewAdminConfigUser()
	c.adminConfig.Users["non-admin"].Repos["test-repo"] = newTestRepoConfig()
	c.users["non-admin"] = models.NewUserConfig()

	// Org-level permissions
	c.adminConfig.Users["org-admin"] = models.NewAdminConfigUser()
	c.users["org-admin"] = models.NewUserConfig()
	c.adminConfig.Users["org-write"] = models.NewAdminConfigUser()
	c.users["org-write"] = models.NewUserConfig()
	c.adminConfig.Users["org-read"] = models.NewAdminConfigUser()
	c.users["org-read"] = models.NewUserConfig()

	// Repo-level permissions
	c.adminConfig.Users["read-user"] = models.NewAdminConfigUser()
	c.users["read-user"] = models.NewUserConfig()
	c.adminConfig.Users["write-user"] = models.NewAdminConfigUser()
	c.users["write-user"] = models.NewUserConfig()
	c.adminConfig.Users["nothing-user"] = models.NewAdminConfigUser()

	c.adminConfig.Users["disabled"] = models.NewAdminConfigUser()
	c.adminConfig.Users["disabled"].Disabled = true
	c.adminConfig.Users["disabled"].Keys = []models.PublicKey{
		mustParsePK("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBx4DYr9m+EnG0tgFsUIZqrDP7pa+vpVXJJ6/PE9J7Ll disabled"),
	}
	c.users["disabled"] = models.NewUserConfig()

	// Basic group
	c.adminConfig.Groups["admins"] = []string{"an-admin"}
	c.adminConfig.Groups["nested-admins"] = []string{"$admins"}

	// Group loop
	c.adminConfig.Groups["loop"] = []string{"$loop1"}
	c.adminConfig.Groups["loop1"] = []string{"$loop2"}
	c.adminConfig.Groups["loop2"] = []string{"$loop1"}

	// Basic org
	c.adminConfig.Orgs["an-org"] = models.NewOrgConfig()
	c.adminConfig.Orgs["an-org"].Admin = []string{"org-admin"}
	c.adminConfig.Orgs["an-org"].Write = []string{"org-write"}
	c.adminConfig.Orgs["an-org"].Read = []string{"org-read"}
	c.adminConfig.Orgs["an-org"].Repos["test-repo"] = newTestRepoConfig()
	c.orgs["an-org"] = models.NewOrgConfig()

	c.adminConfig.Repos["test-repo"] = newTestRepoConfig()

	c.adminConfig.Invites["valid-invite"] = "an-admin"
	c.adminConfig.Invites["user-missing"] = "invalid-user"
	c.adminConfig.Invites["user-disabled"] = "disabled"

	// Force all settings repos to "on"
	c.adminConfig.Options.UserConfigRepos = true
	c.adminConfig.Options.OrgConfig = true
	c.adminConfig.Options.OrgConfigRepos = true

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

func TestAccessLevelStringer(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "None", AccessLevelNone.String())
	assert.Equal(t, "Read", AccessLevelRead.String())
	assert.Equal(t, "Write", AccessLevelWrite.String())
	assert.Equal(t, "Admin", AccessLevelAdmin.String())
	assert.Equal(t, "Unknown(42)", AccessLevel(42).String())
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

		require.Nil(t, err, "Failed to load repo: %q", test.Lookup)

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
	Admin      AccessLevel
	OrgConfig  AccessLevel
	OrgRepo    AccessLevel
	UserConfig AccessLevel
	UserRepo   AccessLevel
	TopLevel   AccessLevel
}

type allImplicitAccessLevels struct {
	Org      AccessLevel
	User     AccessLevel
	TopLevel AccessLevel
}

func lookupAndCheck(t *testing.T, c *Config, u *UserSession, path string, access AccessLevel) {
	repo, err := c.LookupRepoAccess(u, path)
	require.Nil(t, err)
	require.NotNil(t, repo)
	assert.Equal(t, access, repo.Access)
}

func testCheckRepoAccess(t *testing.T, c *Config, u *UserSession, access allRepoAccessLevels) {
	lookupAndCheck(t, c, u, "admin", access.Admin)
	lookupAndCheck(t, c, u, "@an-org", access.OrgConfig)
	lookupAndCheck(t, c, u, "@an-org/test-repo", access.OrgRepo)
	lookupAndCheck(t, c, u, "~non-admin", access.UserConfig)
	lookupAndCheck(t, c, u, "~non-admin/test-repo", access.UserRepo)
	lookupAndCheck(t, c, u, "test-repo", access.TopLevel)
}

func testImplicitRepoAccess(t *testing.T, c *Config, u *UserSession, access allImplicitAccessLevels) {
	prevImplicit := c.adminConfig.Options.ImplicitRepos

	defer func() {
		c.adminConfig.Options.ImplicitRepos = prevImplicit
	}()

	c.adminConfig.Options.ImplicitRepos = true

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
				Admin:      AccessLevelAdmin,
				OrgConfig:  AccessLevelAdmin,
				OrgRepo:    AccessLevelAdmin,
				UserConfig: AccessLevelAdmin,
				UserRepo:   AccessLevelAdmin,
				TopLevel:   AccessLevelAdmin,
			},
			allImplicitAccessLevels{
				Org:      AccessLevelAdmin,
				User:     AccessLevelAdmin,
				TopLevel: AccessLevelAdmin,
			},
		},
		{
			"org-admin",
			allRepoAccessLevels{
				OrgConfig: AccessLevelAdmin,
				OrgRepo:   AccessLevelAdmin,
			},
			allImplicitAccessLevels{
				Org: AccessLevelAdmin,
			},
		},
		{
			"org-write",
			allRepoAccessLevels{
				OrgRepo: AccessLevelWrite,
			},
			allImplicitAccessLevels{
				Org: AccessLevelWrite,
			},
		},
		{
			"org-read",
			allRepoAccessLevels{
				OrgRepo: AccessLevelRead,
			},
			allImplicitAccessLevels{
				Org: AccessLevelRead,
			},
		},
		{
			"non-admin",
			allRepoAccessLevels{
				UserConfig: AccessLevelAdmin,
				UserRepo:   AccessLevelAdmin,
			},
			allImplicitAccessLevels{
				User: AccessLevelAdmin,
			},
		},
		{
			"write-user",
			allRepoAccessLevels{
				OrgRepo:  AccessLevelWrite,
				UserRepo: AccessLevelWrite,
				TopLevel: AccessLevelWrite,
			},
			allImplicitAccessLevels{},
		},
		{
			"read-user",
			allRepoAccessLevels{
				OrgRepo:  AccessLevelRead,
				UserRepo: AccessLevelRead,
				TopLevel: AccessLevelRead,
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
		user, ok := c.adminConfig.Users[test.Username]
		require.True(t, ok)

		session := &UserSession{
			Username: test.Username,
			IsAdmin:  user.IsAdmin,
		}

		testCheckRepoAccess(t, c, session, test.Access)
		testImplicitRepoAccess(t, c, session, test.ImplicitAccess)
	}

	// One special test case to check a repo that doesn't exist.
	repo, err := c.LookupRepoAccess(&UserSession{Username: "an-admin", IsAdmin: true}, "invalid-repo")
	require.Equal(t, ErrRepoDoesNotExist, err)
	require.Nil(t, repo)
}
