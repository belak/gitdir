# go-git-dir

This project was inspired by gitolite, but also includes a built-in ssh server
and some additional flexability. It is not considered stable, but should be
usable enough to experiment with.

The main goal of this project is to enable simple git hosting when a full
solution like Bitbucket, Github, Gitlab, Gitea, etc is not needed. It should
require no additional resources, other than the directory it is pointed at, the
running server, and a git installation.

Thankfully because all the repos are simply stored as bare git repositories, it
should be fairly simple to migrate to or from other git hosting solutions.

## Requirements

Build requirements:

- Go 1.13

Runtime requirements:

- git

## Building

From the root of the source tree, run:

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

### Runtime Config

The runtime config is stored in the "admin" repository. It can be cloned and
modified by any admin on the server. In it you can specify groups (groupings of
users for config or convenience reasons), repos, and orgs (groupings of repos
managed by a person).

Additionally, there are a number of options that can be specified in this file
which change the behavior of the server.

- `user_repos` - allow users to have repos located at ~username/repo-name

There are a number of additional options in the sample config but they have not
been implemented yet.

## Usage

Simply run the built binary with `GITDIR_BASE_DIR` set and start using it!

On first run, go-git-dir will push a commit to the admin repo with a sample
config as well as generated server ssh keys. These can be updated at any time
(even at runtime) but if the server restarts and the keys cannot be loaded, they
will be re-generated.

Note that you will also need to add a user. The following command is a
convenience for adding a user to the admin config.

```
$ go-gitdir --base-dir=/tmp/git add-user --username=belak --pubkey=$HOME/.ssh/id_rsa.pub
```

Note that you will need to manually clone the admin repository (at
`$GITDIR_BASE_DIR/admin/admin`) to add a user as a .yml file in the users dir
and define the admins group before things work as expected.

Example user file:

```
keys:
  - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIIdmFwaB4lwmogg7ggFE8M45Zywx1W3T7dGktY563FM belak@laptop
```

## Repo Creation

For any repo defined in the config, on the first access of that repo (push or
pull), it will be created automatically.
