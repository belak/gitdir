package main

import (
	"context"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type contextKey string

func (c contextKey) String() string {
	return "context key: " + string(c)
}

const (
	contextKeyUser          = contextKey("go-git-dir-user")
	contextKeyLogger        = contextKey("go-git-dir-logger")
	contextKeyAdminSettings = contextKey("go-git-dir-admin-settings")
)

// CtxExtract is a convenience wrapper around the other context convenience
// methods to pull out everything you'd want from a request.
func CtxExtract(ctx context.Context) (*zerolog.Logger, *AdminConfig, *User) {
	return CtxLogger(ctx), CtxSettings(ctx), CtxUser(ctx)
}

// CtxSetUser puts the given User into the ssh.Context.
func CtxSetUser(parent ssh.Context, user *User) {
	parent.SetValue(contextKeyUser, user)
}

// CtxUser pulls the current User out of the context, or AnonymousUser if not
// set.
func CtxUser(ctx context.Context) *User {
	if u, ok := ctx.Value(contextKeyUser).(*User); ok {
		return u
	}

	return AnonymousUser
}

// CtxSetSettings puts the given AdminConfig into the context. This is done so
// an ssh session can contain all the information needed to load the
// repository. It makes it possible to reload the config without messing with
// existing connections.
func CtxSetSettings(parent ssh.Context, ac *AdminConfig) {
	parent.SetValue(contextKeyAdminSettings, ac)
}

// CtxSettings pulls the AdminConfig out of the context, or an empty
// AdminConfig if not found.
func CtxSettings(ctx context.Context) *AdminConfig {
	if ac, ok := ctx.Value(contextKeyAdminSettings).(*AdminConfig); ok {
		return ac
	}

	// A default config should be empty and disallow all access.
	return newAdminConfig()
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
	ctxLog, ok := ctx.Value(contextKeyLogger).(*zerolog.Logger)
	if !ok {
		return &log.Logger
	}

	return ctxLog
}
