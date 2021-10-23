# go-gitdir

[![Go Report Card](https://goreportcard.com/badge/github.com/belak/go-gitdir)](https://goreportcard.com/report/github.com/belak/go-gitdir)
[![Build Status](https://travis-ci.org/belak/go-gitdir.svg?branch=master)](https://travis-ci.org/belak/go-gitdir)

This project makes it incredibly easy to host a secure git server with a config
that can be easily rolled back.

It aims to solve a number of problems other git servers have:

- Requires no external dependencies other than the binary and git
- Stores its configuration in a repo managed by itself
- Doesn't hook into the system's user accounts
- No vendor lock-in - everything is just a bare git repository

## Origins

The main goal of this project is to enable simple git hosting when a full
solution like Bitbucket, Github, Gitlab, Gitea, etc is not needed.

This project was inspired by gitolite and gitosis, but also includes a built-in
ssh server and some additional flexability. It is not considered stable, but
should be usable enough to experiment with.

Thankfully because all the repos are simply stored as bare git repositories, it
should be fairly simple to migrate to or from other git hosting solutions. There
is no vendor lock-in.

## Requirements

Build requirements:

- Go >= 1.13

Runtime requirements:

- git (for git-receive-pack and git-upload-pack)

## Building

Clone the repository somewhere, outside the GOPATH. Then, from the root of the
source tree, run:

```
go build
```

This will create a binary called go-gitdir.

## Running

### Server Config

There are a number of environment variables which can be used to configure your
go-git-dir instance.

The following are required:

- `GITDIR_BASE_DIR` - A directory to store all repositories in. This folder must
  exist when the service starts up.

The following are optional:

- `GITDIR_BIND_ADDR` - The address and port to bind the service to. This
  defaults to `:2222`.
- `GITDIR_LOG_READABLE` - A true value if the log should be human readable
- `GITDIR_LOG_DEBUG` - A true value if debug logging should be enabled
- `GITDIR_ADMIN_USER` - The name of an admin user which the server will ensure
  exists on startup.
- `GITHUB_ADMIN_PUBLIC_KEY` - The contents of a public key which will be added
  to the admin user on startup.

### Runtime Config

The runtime config is stored in the "admin" repository. It can be cloned and
modified by any admin on the server. In it you can specify groups (groupings of
users for config or convenience reasons), repos, and orgs (groupings of repos
managed by a person).

Additionally, there are a number of options that can be specified in this file
which change the behavior of the server.

- `implicit_repos` - allows a user with admin access to that area to create
  repos by simply pushing to them.
- `user_config_keys` - allows users to specify ssh keys in their own config,
  rather than relying on the main admin config.
- `user_config_repos` - allows users to specify repos in their own config,
  rather than relying on the main admin config.
- `org_config_repos` - allows org admins to specify repos in their own config,
  rather than relying on the main admin config.

## Usage

Simply run the built binary with `GITDIR_BASE_DIR` set and start using it!

On first run, gitdir will push a commit to the admin repo with a sample
config as well as generated server ssh keys. These can be updated at any time
(even at runtime) but if the server restarts and the keys cannot be loaded, they
will be re-generated.

If you set `GITDIR_ADMIN_USER` and `GITHUB_ADMIN_PUBLIC_KEY` an admin user will
automatically be added to the config.

If you do not set those environment variables, you will need to manually clone
the admin repository (at `$GITDIR_BASE_DIR/admin/admin`) to add a user to
`config.yml` and set them as an admin.

## Sample Config

Sample admin `config.yml`:

```
users:
  belak:
    is_admin: true
    keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDeQfBUWIqpGXS8xCOg/0RKVOGTnzpIdL7r9wK1/xA52 belak@tmp
    repos:
      personal-gitdir: {}

groups:
  admins:
    - belak

repos:
  go-gitdir:
    public: true

    write:
      - $admins
    read:
      - some-other-user

orgs:
  vault:
    admins:
      - $admins
    write:
      - some-org-user
    read:
      - some-other-org-user

    repos:
      the-vault:
        write:
          - some-repo-access-user

options:
  implicit_repos: false
  user_config_keys: true
  user_config_repos: false
  org_config_repos: false
```

## Repo Creation

All repos defined in the config are created when the config is loaded. At
runtime, if implicit repos are enabled, trying to access a repo where you have
admin access will implicitly create it.
