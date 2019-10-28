package main

import (
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	lock *sync.RWMutex
	s    *ssh.Server
	c    *Config

	// Server level logger
	log zerolog.Logger

	// This represents the internal state of the server.
	settings *AdminConfig
}

func NewServer(c *Config) (*Server, error) {
	var err error

	serv := &Server{
		lock: &sync.RWMutex{},
		log:  log.Logger,
		c:    c,
	}

	serv.s = &ssh.Server{
		Addr:             serv.c.BindAddr,
		Handler:          serv.handleSession,
		PublicKeyHandler: serv.handlePublicKey,
	}

	// This will set serv.settings
	err = serv.Reload()
	if err != nil {
		return nil, err
	}

	return serv, nil
}

func (serv *Server) GetAdminConfig() *AdminConfig {
	serv.lock.RLock()
	defer serv.lock.RUnlock()

	return serv.settings
}

func (serv *Server) Reload() error {
	log.Info().Msg("Reloading")

	var err error

	serv.lock.Lock()
	defer serv.lock.Unlock()

	var pks []PrivateKey

	serv.settings, pks, err = LoadSettings()
	if err != nil {
		return err
	}

	for _, key := range pks {
		signer, err := gossh.NewSignerFromSigner(key)
		if err != nil {
			return err
		}

		serv.s.AddHostKey(signer)
	}

	return nil
}

func (serv *Server) ListenAndServe() error {
	serv.log.Info().Str("port", serv.c.BindAddr).Msg("Starting SSH server")

	return serv.s.ListenAndServe()
}

func (serv *Server) handlePublicKey(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
	slog := CtxLogger(ctx).With().
		Str("remote_user", ctx.User()).
		Str("remote_addr", ctx.RemoteAddr().String()).Logger()

	// Pull the admin config so we can use it when we need it. This will serve
	// as an anchor point, so if the server reloads, we need to keep the old
	// config in place.
	//
	// NOTE: if the user has a long-lived connection and the settings change out
	// from under the user, the connection should be cycled.
	settings := serv.GetAdminConfig()

	remoteUser := ctx.User()

	user, err := settings.GetUserFromKey(PublicKey{incomingKey, ""})
	if err != nil {
		slog.Warn().Err(err).Msg("User not found")
		return false
	}

	// If they weren't the git user make sure their username matches their key.
	if remoteUser != serv.c.GitUser && remoteUser != user.Username {
		slog.Warn().Msg("Key belongs to different user")
		return false
	}

	// Update the context with what we discovered
	CtxSetUser(ctx, user)
	CtxSetSettings(ctx, settings)
	CtxSetLogger(ctx, &slog)

	// Config will never change, so we can pull this directly from the server.
	CtxSetConfig(ctx, serv.c)

	return true
}

func (serv *Server) handleSession(s ssh.Session) {
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

	// If the user doesn't provide any arguments, we want to run the internal
	// whoami command.
	if len(cmd) == 0 {
		cmd = []string{"whoami"}
	}

	// Add the command to the logger
	tmpLog := slog.With().Str("cmd", cmd[0]).Logger()
	slog = &tmpLog
	ctx = WithLogger(ctx, slog)

	var exit int

	switch cmd[0] {
	case "whoami":
		exit = cmdWhoami(ctx, s, cmd)
	case "git-receive-pack":
		exit = serv.cmdRepoAction(ctx, s, cmd, AccessTypeWrite)
	case "git-upload-pack":
		exit = serv.cmdRepoAction(ctx, s, cmd, AccessTypeRead)
	default:
		exit = cmdNotFound(ctx, s, cmd)
	}

	slog.Info().Int("return_code", exit).Msg("Return code")
	_ = s.Exit(exit)
}

func (c *Config) LookupRepo(repoPath string) (RepoLookup, error) {
	lookup, err := ParseRepo(c, repoPath)
	if err != nil {
		return nil, err
	}

	return lookup, nil
}
