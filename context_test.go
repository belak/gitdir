package gitdir

import (
	"context"
	"testing"

	"github.com/belak/go-gitdir/models"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestContextKey(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		Input    contextKey
		Base     string
		Expected string
	}{
		{
			contextKey("hello world"),
			"hello world",
			"Context key: hello world",
		},
		{
			contextKeyConfig,
			"gitdir-config",
			"Context key: gitdir-config",
		},
		{
			contextKeyUser,
			"gitdir-user",
			"Context key: gitdir-user",
		},
		{
			contextKeyLogger,
			"gitdir-logger",
			"Context key: gitdir-logger",
		},
		{
			contextKeyPublicKey,
			"gitdir-public-key",
			"Context key: gitdir-public-key",
		},
	}

	baseCtx := context.Background()

	for _, test := range tests {
		assert.Equal(t, test.Expected, test.Input.String())

		ctx := context.WithValue(baseCtx, test.Input, "hello world")

		// Make sure you can't pull values out with the raw string
		assert.Nil(t, ctx.Value(test.Base))

		// Assert values come out properly
		assert.Equal(t, "hello world", ctx.Value(test.Input))
	}
}

func TestCtxExtract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	logger, config, user := CtxExtract(ctx)

	assert.Equal(t, &log.Logger, logger)
	assert.Equal(t, newConfig(), config)
	assert.Equal(t, AnonymousUser, user)
}

func TestCtxSetConfig(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestCtxConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Check the default value
	config := CtxConfig(ctx)
	assert.Equal(t, newConfig(), config)

	// Check that when we set a value, this properly extracts it.
	config = newConfig()
	ctx = context.WithValue(ctx, contextKeyConfig, config)
	assert.Equal(t, config, CtxConfig(ctx))
}

func TestCtxSetUser(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestCtxUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Check the default value
	assert.Equal(t, AnonymousUser, CtxUser(ctx))

	// Check that when we set a value, this properly extracts it.
	user := &User{
		Username: "belak",
		IsAdmin:  true,
	}
	ctx = context.WithValue(ctx, contextKeyUser, user)
	assert.Equal(t, user, CtxUser(ctx))
}

func TestCtxSetLogger(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestWithLogger(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestCtxLogger(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Check the default value
	assert.Equal(t, &log.Logger, CtxLogger(ctx))

	// Check that when we set a value, this properly extracts it.
	logger := log.With().Str("hello", "world").Logger()
	ctx = context.WithValue(ctx, contextKeyLogger, &logger)
	assert.Equal(t, &logger, CtxLogger(ctx))
}

func TestCtxSetPublicKey(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestCtxPublicKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Check the default value
	assert.Nil(t, CtxPublicKey(ctx))

	// Check that when we set a value, this properly extracts it.
	pk := &models.PublicKey{}
	ctx = context.WithValue(ctx, contextKeyPublicKey, pk)
	assert.Equal(t, pk, CtxPublicKey(ctx))
}
