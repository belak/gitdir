package main

// TODO: update sample config
const sampleConfig = `# Users define the users who have access to the service. They will need an
# SSH key or invite added to their user account before they can access the
# server.
users:
  some-user: {}
  belak:
    is_admin: true

# Groups can be used in any place a normal user could be used. They are prefixed
# with a $, so the admins group could be accessed with $admins. Groups can be
# defined recursively, but they cannot have loops.
groups:
  admins:
    - belak

# Permissions for top-level repos
repos:
  sample:
    write:
      - $admins

# Org repos are accessible at @org-name/repo. Note that if admins is not
# specified, it will default to users with is_admin. By default, all members
# of an org will have read-write access to repos. This can be changed with the
# read and write keys.
orgs:
  sample:
    # Set permission levels for users in an org. Note that each level implies
    # the previous, so admin also has write and read permissions. In short the
    # permissions do the following:
    #
    # - admin: if org configuration is enabled, admins can read and write the
    #   org-level config repo (located at @org-name)
    # - write: repo write
    # - read: repo read
    admin:
      - $admins
    write:
      - $admins
    read:
      - $admins

    # Map of repos with added permissions.
    repos:
      test-repo:
        write:
          - belak
        read:
          - some-user

  vault: {}

options:
  # Allow users to define repos under their user accounts. Note that those
  # repos will be accessible at ~username/repo rather than the more
  # traditional username/repo. The config is located at ~username.
  user_config_repos: false

  # Allow users to create user repos or org-admins to create org repos
  # simply by cloning or pushing to them.
  implicit_repos: false

  # Allow config at the org level. Note that if this is allowed, the root config
  # will be merged with the org config, but the root config will take precidence
  # for any values that are set. These repos will be accessible at @org-name
  org_config: false
  org_config_repos: false
  org_config_users: false
`
