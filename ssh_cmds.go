package main

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	"github.com/gliderlabs/ssh"
)

func cmdWhoami(ctx context.Context, s ssh.Session, cmd []string) int {
	user := CtxUser(ctx)
	_ = writeStringFmt(s, "logged in as %s\r\n", user.Username)
	return 0
}

func cmdNotFound(ctx context.Context, s ssh.Session, cmd []string) int {
	_ = writeStringFmt(s.Stderr(), "command %q not found\r\n", cmd[0])
	return 1
}

// TODO: don't bother currying here
func (serv *Server) cmdRepoAction(ctx context.Context, s ssh.Session, cmd []string, access AccessType) int {
	if len(cmd) != 2 {
		_ = writeStringFmt(s.Stderr(), "Missing repo name argument")
		return 1
	}

	log, config, settings, user := CtxExtract(ctx)

	// Sanitize the repo name
	//   - Trim all slashes from beginning and end
	//   - Add a root slash (so path.Clean works correctly)
	//   - path.Clean
	//   - Remove the initial slash
	//   - Sanitize the name
	repoName := sanitize(path.Clean("/" + strings.Trim(cmd[1], "/"))[1:])

	repo, err := config.LookupRepo(repoName)
	if err != nil {
		return -1
	}

	if !repo.IsValid(settings) {
		_ = writeStringFmt(s.Stderr(), "Repo does not exist")
		return -1
	}

	if !repo.UserHasAccess(settings, user, access) {
		_ = writeStringFmt(s.Stderr(), "Permission denied")
		return -1
	}

	returnCode := runCommand(log, s, []string{cmd[0], filepath.FromSlash(repo.Path())})

	// Reload the server config if a config repo was changed.
	if access == AccessTypeWrite {
		switch repo.(type) {
		case *repoLookupAdmin, *repoLookupOrgConfig, *repoLookupUserConfig:
			err = serv.Reload()
			if err != nil {
				_ = writeStringFmt(s.Stderr(), "Error when reloading config: %s", err)
			}
		}
	}

	return returnCode
}
