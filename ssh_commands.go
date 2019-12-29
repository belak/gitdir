package gitdir

import (
	"context"
	"path"
	"strings"

	"github.com/gliderlabs/ssh"

	"github.com/belak/go-gitdir/internal/git"
)

func cmdWhoami(ctx context.Context, s ssh.Session, cmd []string) int { //nolint:interfacer
	user := CtxUser(ctx)
	_ = writeStringFmt(s, "logged in as %s\r\n", user.Username)

	return 0
}

func cmdNotFound(ctx context.Context, s ssh.Session, cmd []string) int {
	_ = writeStringFmt(s.Stderr(), "command %q not found\r\n", cmd[0])
	return 1
}

func (serv *Server) cmdRepoAction(ctx context.Context, s ssh.Session, cmd []string, access AccessLevel) int {
	if len(cmd) != 2 {
		_ = writeStringFmt(s.Stderr(), "Missing repo name argument\r\n")
		return 1
	}

	log, config, user := CtxExtract(ctx)
	pk := CtxPublicKey(ctx)

	// Sanitize the repo name
	//   - Trim all slashes from beginning and end
	//   - Add a root slash (so path.Clean works correctly)
	//   - path.Clean
	//   - Remove the initial slash
	//   - Sanitize the name
	repoName := sanitize(path.Clean("/" + strings.Trim(cmd[1], "/"))[1:])

	// Repo does not exist and permission checks should give the same error, so
	// information about what repos are defined is not leaked.
	repo, err := config.LookupRepoAccess(user, repoName)
	if err != nil {
		_ = writeStringFmt(s.Stderr(), "Repo does not exist\r\n")
		return -1
	}

	if repo.Access < access {
		_ = writeStringFmt(s.Stderr(), "Repo does not exist\r\n")
		return -1
	}

	// Because we check ImplicitRepos earlier, if they have admin access, it's
	// safe to ensure this repo exists.
	if repo.Access >= AccessLevelAdmin {
		_, err = git.EnsureRepo(serv.baseConfig.BaseDir, repo.Path())
		if err != nil {
			return -1
		}
	}

	returnCode := runCommand(log, serv.baseConfig.BaseDir, s, []string{cmd[0], repo.Path()}, []string{
		"GITDIR_BASE_DIR=" + serv.baseConfig.BaseDir,
		"GITDIR_HOOK_REPO_PATH=" + repoName,
		"GITDIR_HOOK_PUBLIC_KEY=" + pk.String(),
		"GITDIR_LOG_FORMAT=console",
	})

	// Reload the server config if a config repo was changed.
	if access == AccessLevelWrite {
		switch repo.Type {
		case RepoTypeAdmin, RepoTypeOrgConfig, RepoTypeUserConfig:
			err = serv.Reload()
			if err != nil {
				_ = writeStringFmt(s.Stderr(), "Error when reloading config: %s\r\n", err)
			}
		}
	}

	return returnCode
}
