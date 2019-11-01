package gitdir

import (
	"context"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type contextKey string

func (c contextKey) String() string {
	return "Context key: " + string(c)
}

const (
	contextKeyUser   = contextKey("gitdir-user")
	contextKeyLogger = contextKey("gitdir-logger")
)

// CtxExtract is a convenience wrapper around the other context convenience
// methods to pull out everything you'd want from a request.
func CtxExtract(ctx context.Context) (*zerolog.Logger, *User) {
	return CtxLogger(ctx), CtxUser(ctx)
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
