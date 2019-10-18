package main

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	"github.com/gliderlabs/ssh"
)

type sshCommand func(ctx context.Context, s ssh.Session, cmd []string) int

func cmdWhoami(ctx context.Context, s ssh.Session, cmd []string) int {
	user := CtxUser(ctx)
	_ = writeStringFmt(s, "logged in as %s\r\n", user.Username)
	return 0
}

func cmdNotFound(ctx context.Context, s ssh.Session, cmd []string) int {
	_ = writeStringFmt(s.Stderr(), "command %q not found\r\n", cmd[0])
	return 1
}

func (serv *server) cmdRepoAction(access accessType) sshCommand {
	return func(ctx context.Context, s ssh.Session, cmd []string) int {
		if len(cmd) != 2 {
			_ = writeStringFmt(s.Stderr(), "Missing repo name argument")
			return 1
		}

		log, _, user := CtxExtract(ctx)

		// Sanitize the repo name
		//   - Trim all slashes from beginning and end
		//   - Add a root slash (so path.Clean works correctly)
		//   - path.Clean
		//   - Remove the initial slash
		//   - Sanitize the name
		repoName := sanitize(path.Clean("/" + strings.Trim(cmd[1], "/"))[1:])

		repo, err := serv.LookupRepo(repoName, user, access)
		if err != nil {
			return -1
		}

		returnCode := runCommand(log, s, []string{cmd[0], filepath.FromSlash(repo.Path)})

		switch repo.Type {
		case RepoTypeAdmin, RepoTypeOrgConfig, RepoTypeUserConfig:
			// TODO: show error
			err = serv.Reload()
			if err != nil {
				return -1
			}
		}

		return returnCode
	}
}
