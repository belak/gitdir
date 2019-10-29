package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v3"
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

func (serv *Server) AcceptInvite(invite string, key PublicKey) bool { //nolint:funlen
	serv.lock.Lock()
	defer serv.lock.Unlock()

	// It's expensive to lock, so we need to do the fast stuff first and bail as
	// early as possible if it's not valid.

	// Step 1: Look up the invite
	username, ok := serv.settings.Invites[invite]
	if !ok {
		return false
	}

	fmt.Println("found user")

	adminRepo, err := EnsureRepo("admin/admin", true)
	if err != nil {
		log.Warn().Err(err).Msg("Admin repo doesn't exist")
		return false
	}

	err = adminRepo.UpdateFile("config.yml", func(data []byte) ([]byte, error) {
		rootNode, _, err := ensureSampleConfigYaml(data) //nolint:govet
		if err != nil {
			return nil, err
		}

		// We can assume the config file is in a valid format because of
		// ensureSampleConfig
		targetNode := rootNode.Content[0]

		// Step 2: Ensure the user exists and is not disabled.
		usersVal := yamlLookupVal(targetNode, "users")
		userVal, _ := yamlEnsureKey(usersVal, username, &yaml.Node{Kind: yaml.MappingNode}, "", false)
		_ = yamlRemoveKey(userVal, "disabled")

		// Step 3: Add the key to the user
		keysVal, _ := yamlEnsureKey(userVal, "keys", &yaml.Node{Kind: yaml.SequenceNode}, "", false)
		keysVal.Content = append(keysVal.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key.MarshalAuthorizedKey(),
		})

		// Step 4: Remove the invite (and any others this user owns)
		var staleInvites []string
		invitesVal := yamlLookupVal(targetNode, "invites")

		for i := 0; i+1 < len(invitesVal.Content); i += 2 {
			if invitesVal.Content[i+1].Value == username {
				staleInvites = append(staleInvites, invitesVal.Content[i].Value)
			}
		}

		for _, val := range staleInvites {
			yamlRemoveKey(invitesVal, val)
		}

		// Step 5: Re-encode back to yaml
		data, err = yamlEncode(rootNode)
		return data, err
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to update config")
		return false
	}

	err = adminRepo.Commit("Added "+username+" from invite "+invite, nil)
	if err != nil {
		return false
	}

	err = serv.reloadInternal()

	// The invite was successfully accepted if the server reloaded properly.
	return err == nil
}

func (serv *Server) Reload() error {
	serv.lock.Lock()
	defer serv.lock.Unlock()

	return serv.reloadInternal()
}

func (serv *Server) reloadInternal() error {
	log.Info().Msg("Reloading")

	var err error

	var pks []PrivateKey

	serv.settings, pks, err = LoadAdminConfig()
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

	if strings.HasPrefix(remoteUser, settings.Options.InvitePrefix) {
		invite := remoteUser[len(settings.Options.InvitePrefix):]

		// Try to accept the invite. If this fails, bail out. Otherwise,
		// continue looking up the user as normal.
		if ok := serv.AcceptInvite(invite, PublicKey{incomingKey, ""}); !ok {
			return false
		}

		// If it succeeded, we actually need to pull the refreshed admin config
		// so the new user shows up.
		settings = serv.GetAdminConfig()
	}

	user, err := settings.GetUserFromKey(PublicKey{incomingKey, ""})
	if err != nil {
		slog.Warn().Err(err).Msg("User not found")
		return false
	}

	// If they weren't the git user make sure their username matches their key.
	if remoteUser != settings.Options.GitUser && remoteUser != user.Username {
		slog.Warn().Msg("Key belongs to different user")
		return false
	}

	// Update the context with what we discovered
	CtxSetUser(ctx, user)
	CtxSetSettings(ctx, settings)
	CtxSetLogger(ctx, &slog)

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
