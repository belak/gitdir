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
	contextKeyConfig        = contextKey("go-git-dir-config")
	contextKeyLogger        = contextKey("go-git-dir-logger")
	contextKeyAdminSettings = contextKey("go-git-dir-admin-settings")
)

func CtxExtract(ctx context.Context) (*zerolog.Logger, *Config, *AdminConfig, *User) {
	return CtxLogger(ctx), CtxConfig(ctx), CtxSettings(ctx), CtxUser(ctx)
}

func CtxSetUser(parent ssh.Context, user *User) {
	parent.SetValue(contextKeyUser, user)
}

func CtxUser(ctx context.Context) *User {
	if u, ok := ctx.Value(contextKeyUser).(*User); ok {
		return u
	}

	return AnonymousUser
}

func CtxSetSettings(parent ssh.Context, a *AdminConfig) {
	parent.SetValue(contextKeyAdminSettings, a)
}

func CtxSettings(ctx context.Context) *AdminConfig {
	if s, ok := ctx.Value(contextKeyAdminSettings).(*AdminConfig); ok {
		return s
	}

	// A default config should be empty and disallow all access.
	return &AdminConfig{}
}

func WithLogger(parent context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(parent, contextKeyLogger, logger)
}

func CtxSetLogger(parent ssh.Context, logger *zerolog.Logger) {
	parent.SetValue(contextKeyLogger, logger)
}

func CtxLogger(ctx context.Context) *zerolog.Logger {
	ctxLog, ok := ctx.Value(contextKeyLogger).(*zerolog.Logger)
	if !ok {
		return &log.Logger
	}

	return ctxLog
}

func CtxSetConfig(parent ssh.Context, config *Config) {
	parent.SetValue(contextKeyConfig, config)
}

func CtxConfig(ctx context.Context) *Config {
	return ctx.Value(contextKeyConfig).(*Config)
}
