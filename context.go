package gitdir

import (
	"context"

	"github.com/belak/go-gitdir/models"
	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

type contextKey string

func (c contextKey) String() string {
	return "Context key: " + string(c)
}

const (
	contextKeyConfig    = contextKey("gitdir-config")
	contextKeyUser      = contextKey("gitdir-user")
	contextKeyLogger    = contextKey("gitdir-logger")
	contextKeyPublicKey = contextKey("gitdir-public-key")
)

// CtxExtract is a convenience wrapper around the other context convenience
// methods to pull out everything you'd want from a request.
func CtxExtract(ctx context.Context) (*zerolog.Logger, *Config, *UserSession) {
	return CtxLogger(ctx), CtxConfig(ctx), CtxUserSession(ctx)
}

// CtxSetConfig puts the given Config into the ssh.Context.
func CtxSetConfig(parent ssh.Context, config *Config) {
	parent.SetValue(contextKeyConfig, config)
}

// CtxConfig pulls the current Config out of the context, or a blank Config if
// not set.
func CtxConfig(ctx context.Context) *Config {
	if c, ok := ctx.Value(contextKeyConfig).(*Config); ok {
		return c
	}

	// If it doesn't exist, return a new empty config for safety.
	return NewConfig(memfs.New())
}

// CtxSetUser puts the given User into the ssh.Context.
func CtxSetUserSession(parent ssh.Context, user *UserSession) {
	parent.SetValue(contextKeyUser, user)
}

// CtxUser pulls the current User out of the context, or AnonymousUser if not
// set.
func CtxUserSession(ctx context.Context) *UserSession {
	if u, ok := ctx.Value(contextKeyUser).(*UserSession); ok {
		return u
	}

	return AnonymousUserSession
}

// WithLogger takes a parent context and a logger and returns a new context
// with that logger.
func WithLogger(parent context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(parent, contextKeyLogger, logger)
}

// CtxSetLogger puts the given logger into the ssh.Context.
func CtxSetLogger(parent ssh.Context, logger *zerolog.Logger) {
	parent.SetValue(contextKeyLogger, logger)
}

// CtxLogger pulls the logger out of the context, or the default logger if not
// found.
func CtxLogger(ctx context.Context) *zerolog.Logger {
	if ctxLog, ok := ctx.Value(contextKeyLogger).(*zerolog.Logger); ok {
		return ctxLog
	}

	return &log.Logger
}

// CtxSetPublicKey puts the given public key into the ssh.Context.
func CtxSetPublicKey(parent ssh.Context, pk *models.PublicKey) {
	parent.SetValue(contextKeyPublicKey, pk)
}

// CtxPublicKey pulls the public key out of the context, or nil if not found.
func CtxPublicKey(ctx context.Context) *models.PublicKey {
	if pk, ok := ctx.Value(contextKeyPublicKey).(*models.PublicKey); ok {
		return pk
	}

	return nil
}
