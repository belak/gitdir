package gitdir

import (
	"net"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-billy.v4"

	"github.com/belak/go-gitdir/models"
)

// Server represents a gitdir server.
type Server struct {
	lock *sync.RWMutex

	Addr string

	// Internal state
	log    zerolog.Logger
	fs     billy.Filesystem
	config *Config
	ssh    *ssh.Server
}

// NewServer configures a new gitdir server and attempts to load the config
// from the admin repo.
func NewServer(fs billy.Filesystem) (*Server, error) {
	serv := &Server{
		lock: &sync.RWMutex{},
		log:  log.Logger,
		fs:   fs,
	}

	serv.ssh = &ssh.Server{
		Handler:          serv.handleSession,
		PublicKeyHandler: serv.handlePublicKey,
	}

	// This will set serv.settings
	err := serv.Reload()
	if err != nil {
		return nil, err
	}

	return serv, nil
}

// Reload reloads the server config in a thread-safe way.
func (serv *Server) Reload() error {
	serv.lock.Lock()
	defer serv.lock.Unlock()

	// Create a new config object
	config := NewConfig(serv.fs)

	// Load the config from master
	err := config.Load("")
	if err != nil {
		return err
	}

	serv.config = config

	// Load all ssh keys into the actual ssh server.
	for _, key := range serv.config.pks {
		signer, err := gossh.NewSignerFromSigner(key)
		if err != nil {
			return err
		}

		serv.ssh.AddHostKey(signer)
	}

	return nil
}

// Serve listens on the given listener for new SSH connections.
func (serv *Server) Serve(l net.Listener) error {
	return serv.ssh.Serve(l)
}

// ListenAndServe listens on the Addr set on the server struct for new SSH
// connections.
func (serv *Server) ListenAndServe() error {
	serv.log.Info().Str("port", serv.Addr).Msg("Starting SSH server")

	// Because we're using ListenAndServe, we need to copy in the bind address.
	serv.ssh.Addr = serv.Addr

	return serv.ssh.ListenAndServe()
}

// GetAdminConfig returns the current admin config in a thread-safe manner. The
// config should not be modified.
func (serv *Server) GetAdminConfig() *Config {
	serv.lock.RLock()
	defer serv.lock.RUnlock()

	return serv.config
}

func (serv *Server) handlePublicKey(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
	slog := CtxLogger(ctx).With().
		Str("remote_user", ctx.User()).
		Str("remote_addr", ctx.RemoteAddr().String()).Logger()

	remoteUser := ctx.User()

	config := serv.GetAdminConfig()

	pk := models.PublicKey{PublicKey: incomingKey}

	/*
		if strings.HasPrefix(remoteUser, settings.Options.InvitePrefix) {
			invite := remoteUser[len(settings.Options.InvitePrefix):]

			// Try to accept the invite. If this fails, bail out. Otherwise,
			// continue looking up the user as normal.
			if ok := serv.AcceptInvite(invite, models.PublicKey{incomingKey, ""}); !ok {
				return false
			}

			// If it succeeded, we actually need to pull the refreshed admin config
			// so the new user shows up.
			settings = serv.GetAdminConfig()
		}
	*/

	user, err := config.LookupUserFromPublicKey(pk, remoteUser)
	if err != nil {
		slog.Warn().Err(err).Msg("User not found")
		return false
	}

	// Update the context with what we discovered
	CtxSetUserSession(ctx, user)
	CtxSetConfig(ctx, config)
	CtxSetLogger(ctx, &slog)
	CtxSetPublicKey(ctx, &pk)

	return true
}

func (serv *Server) handleSession(s ssh.Session) {
	ctx := s.Context()

	// Pull a logger for the session
	slog := CtxLogger(ctx)

	defer func() {
		// Note that we can't pass in slog as an argument because that would
		// result in the value getting captured and we want to be able to
		// annotate this with new values.
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
		exit = serv.cmdRepoAction(ctx, s, cmd, AccessLevelWrite)
	case "git-upload-pack":
		exit = serv.cmdRepoAction(ctx, s, cmd, AccessLevelRead)
	default:
		exit = cmdNotFound(ctx, s, cmd)
	}

	slog.Info().Int("return_code", exit).Msg("Return code")
	_ = s.Exit(exit)
}
