package main

import (
	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	extEd25519 "golang.org/x/crypto/ed25519"
	gossh "golang.org/x/crypto/ssh"
)

type server struct {
	s   *ssh.Server
	log zerolog.Logger
	c   *Config

	repo *AdminRepo

	sshCommands map[string]sshCommand
}

func newServer(c *Config) (*server, error) {
	serv := &server{
		log: log.Logger,
		c:   c,
	}

	serv.s = &ssh.Server{
		Addr:             serv.c.BindAddr,
		Handler:          serv.handleSession,
		PublicKeyHandler: serv.handlePublicKey,
	}

	// Set up a mapping of what ssh commands call what functions.
	serv.sshCommands = map[string]sshCommand{
		"whoami":           cmdWhoami,
		"git-receive-pack": serv.cmdRepoAction(accessTypeWrite),
		"git-upload-pack":  serv.cmdRepoAction(accessTypeRead),
	}

	// Look up the repository
	repoLookup, err := LookupRepo(c, "admin")
	if err != nil {
		return nil, err
	}

	// Make sure it exists - we can't use EnsureRepo because that requires us to
	// have an admin repo.
	rawRepo, err := repoLookup.Ensure()
	if err != nil {
		return nil, err
	}

	serv.repo, err = OpenAdminRepo(rawRepo)
	if err != nil {
		return nil, err
	}

	err = serv.reloadInternal()
	if err != nil {
		return nil, err
	}

	return serv, nil
}

func (serv *server) Reload() error {
	err := serv.repo.Reload()
	if err != nil {
		return err
	}

	return serv.reloadInternal()
}

func (serv *server) reloadInternal() error {
	keys, err := serv.repo.GetServerKeys()
	if err != nil {
		return err
	}

	rsaSigner, err := gossh.NewSignerFromSigner(keys.RSA)
	if err != nil {
		return err
	}

	// There are some oddities with how keys are handled here. Because ed25519
	// was a separate package for a while and that's what the ssh package
	// depends on, we need to use that, even though it's just a type alias as of
	// Go 1.13 anyway.
	ed25519Key := extEd25519.PrivateKey(keys.Ed25519)
	ed25519Signer, err := gossh.NewSignerFromSigner(ed25519Key)
	if err != nil {
		return err
	}

	// Add the loaded keys to the server
	serv.s.AddHostKey(rsaSigner)
	serv.s.AddHostKey(ed25519Signer)

	return nil
}

func (serv *server) ListenAndServe() error {
	serv.log.Info().Str("port", serv.c.BindAddr).Msg("Starting SSH server")

	return serv.s.ListenAndServe()
}

func (serv *server) handlePublicKey(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
	slog := CtxLogger(ctx).With().
		Str("remote_user", ctx.User()).
		Str("remote_addr", ctx.RemoteAddr().String()).Logger()

	remoteUser := ctx.User()

	user, err := serv.repo.GetUserFromKey(incomingKey)
	if err != nil {
		slog.Warn().Err(err).Msg("User not found")
		return false
	}

	// If they weren't the git user make sure their username matches their key.
	if remoteUser != serv.c.GitUser && remoteUser != user.Username {
		slog.Warn().Msg("Key belongs to different user")
		return false
	}

	CtxSetUser(ctx, user)
	CtxSetConfig(ctx, serv.c)
	CtxSetLogger(ctx, &slog)

	return true
}

func (serv *server) handleSession(s ssh.Session) {
	ctx := s.Context()

	// Pull a logger for the session
	slog := CtxLogger(ctx)

	defer func() {
		// Note that we can't pass in slog as an argument because that would
		// result in the value getting captured and we want to be able to update
		// this.
		handlePanic(slog)
	}()

	slog.Info().Msg("Starting session")
	defer slog.Info().Msg("Session closed")

	cmd := s.Command()

	// If the user doesn't provide any arguments, we want to run the
	// internal whoami command.
	if len(cmd) == 0 {
		cmd = []string{"whoami"}
	}

	var exit int = 1

	tmpLog := slog.With().Str("cmd", cmd[0]).Logger()
	slog = &tmpLog
	ctx = WithLogger(ctx, slog)

	if cb := serv.sshCommands[cmd[0]]; cb != nil {
		slog.Info().Msg("Running command")
		exit = cb(ctx, s, cmd)
	} else {
		slog.Info().Msg("Command not found")
		exit = cmdNotFound(ctx, s, cmd)
	}

	slog.Info().Int("return_code", exit).Msg("Return code")
	s.Exit(exit)
}
