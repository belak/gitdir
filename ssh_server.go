package gitdir

import (
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/belak/go-gitdir/models"
)

// Server represents a gitdir server.
type Server struct {
	lock *sync.RWMutex
	s    *ssh.Server
	c    *Config

	// Internal state
	log zerolog.Logger

	settings   *models.AdminConfig
	users      map[string]*models.UserConfig
	orgs       map[string]*models.OrgConfig
	publicKeys map[string]string
}

// NewServer configures a new gitdir server and attempts to load the config
// from the admin repo.
func NewServer(c *Config) (*Server, error) {
	var err error

	serv := &Server{
		lock: &sync.RWMutex{},
		log:  log.Logger,
		c:    c,

		users:      make(map[string]*models.UserConfig),
		orgs:       make(map[string]*models.OrgConfig),
		publicKeys: make(map[string]string),
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

// Reload reloads the server config in a thread-safe way.
func (serv *Server) Reload() error {
	serv.lock.Lock()
	defer serv.lock.Unlock()

	return serv.reloadInternal()
}

// ListenAndServe listens on the BindAddr given in the config.
func (serv *Server) ListenAndServe() error {
	serv.log.Info().Str("port", serv.c.BindAddr).Msg("Starting SSH server")

	return serv.s.ListenAndServe()
}

func (serv *Server) handlePublicKey(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
	slog := CtxLogger(ctx).With().
		Str("remote_user", ctx.User()).
		Str("remote_addr", ctx.RemoteAddr().String()).Logger()

	remoteUser := ctx.User()

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

	user, err := serv.LookupUserFromKey(models.PublicKey{PublicKey: incomingKey}, remoteUser)
	if err != nil {
		slog.Warn().Err(err).Msg("User not found")
		return false
	}

	// Update the context with what we discovered
	CtxSetUser(ctx, user)
	CtxSetLogger(ctx, &slog)

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
		exit = serv.cmdRepoAction(ctx, s, cmd, AccessTypeWrite)
	case "git-upload-pack":
		exit = serv.cmdRepoAction(ctx, s, cmd, AccessTypeRead)
	default:
		exit = cmdNotFound(ctx, s, cmd)
	}

	slog.Info().Int("return_code", exit).Msg("Return code")
	_ = s.Exit(exit)
}
