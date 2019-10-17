package main

import (
	"context"
	"path"
	"strings"

	"github.com/gliderlabs/ssh"
)

type sshCommand func(ctx context.Context, s ssh.Session, cmd []string) int

func cmdWhoami(ctx context.Context, s ssh.Session, cmd []string) int {
	user := CtxUser(ctx)
	writeStringFmt(s, "logged in as %s\r\n", user.Username)
	return 0
}

func cmdNotFound(ctx context.Context, s ssh.Session, cmd []string) int {
	writeStringFmt(s, "command %q not found\r\n", cmd[0])
	return 1
}

func (serv *server) cmdRepoAction(access accessType) sshCommand {
	return func(ctx context.Context, s ssh.Session, cmd []string) int {
		if len(cmd) != 2 {
			writeStringFmt(s.Stderr(), "Missing repo name argument")
			return 1
		}

		log, config, user := CtxExtract(ctx)

		// Sanitize the repo name
		//   - Trim all slashes from beginning and end
		//   - Add a root slash (so path.Clean works correctly)
		//   - path.Clean
		//   - Remove the initial slash
		//   - Sanitize the name
		repoName := sanitize(path.Clean("/" + strings.Trim(cmd[1], "/"))[1:])

		repo, err := EnsureRepo(config, serv.repo, repoName)
		if err != nil {
			writeStringFmt(s.Stderr(), "Invalid repo format %q\r\n", cmd[1])
			return -1
		}

		// If the user isn't allowed access, bail before we even bother checking
		// if the repo exists.
		if !serv.repo.settings.UserHasRepoAccess(user, repo, access) {
			return -1
		}

		returnCode := runCommand(log, s, []string{cmd[0], repo.Path})

		switch repo.Type {
		case repoTypeAdmin, repoTypeOrgConfig, repoTypeUserConfig:
			// TODO: show error
			err = serv.Reload()
			if err != nil {
				return -1
			}
		}

		return returnCode
	}
}
