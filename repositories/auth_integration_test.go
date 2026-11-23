//go:build integration

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenLifecycle(t *testing.T) {
	store := newTestStore(t)
	users, repo := NewUserRepository(store), NewAuthRepository(store)
	ctx := context.Background()
	u, err := users.Create(ctx, newUser("ann@example.com"))
	require.NoError(t, err)

	require.NoError(t, repo.CreateRefreshToken(ctx, u.ID, "h1", time.Now().Add(time.Hour)))
	assert.ErrorIs(t, repo.CreateRefreshToken(ctx, u.ID, "h1", time.Now().Add(time.Hour)), ErrConflict, "hash unique")

	tok, err := repo.ConsumeRefreshToken(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, u.ID, tok.UserID)
	_, err = repo.ConsumeRefreshToken(ctx, "h1")
	assert.ErrorIs(t, err, ErrNotFound, "single use")

	require.NoError(t, repo.CreateRefreshToken(ctx, u.ID, "expired", time.Now().Add(-time.Minute)))
	_, err = repo.ConsumeRefreshToken(ctx, "expired")
	assert.ErrorIs(t, err, ErrNotFound, "expired")

	require.NoError(t, repo.CreateRefreshToken(ctx, u.ID, "h2", time.Now().Add(time.Hour)))
	require.NoError(t, repo.CreateRefreshToken(ctx, u.ID, "h3", time.Now().Add(time.Hour)))
	require.NoError(t, repo.RevokeRefreshToken(ctx, "00000000-0000-4000-8000-000000000000", "h2"))
	_, err = repo.ConsumeRefreshToken(ctx, "h2")
	assert.NoError(t, err, "revoke is scoped to the owner")
	require.NoError(t, repo.RevokeRefreshToken(ctx, u.ID, "h3"))
	_, err = repo.ConsumeRefreshToken(ctx, "h3")
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.CreateRefreshToken(ctx, u.ID, "h4", time.Now().Add(time.Hour)))
	require.NoError(t, repo.RevokeAllRefreshTokens(ctx, u.ID))
	_, err = repo.ConsumeRefreshToken(ctx, "h4")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPasswordResetLifecycle(t *testing.T) {
	store := newTestStore(t)
	users, repo := NewUserRepository(store), NewAuthRepository(store)
	ctx := context.Background()
	u, err := users.Create(ctx, newUser("ann@example.com"))
	require.NoError(t, err)

	_, err = repo.GetLatestPasswordReset(ctx, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.CreatePasswordReset(ctx, u.ID, "c1", time.Now().Add(time.Minute)))
	require.NoError(t, repo.CreatePasswordReset(ctx, u.ID, "c2", time.Now().Add(time.Minute)))
	r, err := repo.GetLatestPasswordReset(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "c2", r.CodeHash, "a new reset replaces the old one")

	n, err := repo.RecordResetAttempt(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	require.NoError(t, repo.MarkPasswordResetUsed(ctx, r.ID))
	assert.ErrorIs(t, repo.MarkPasswordResetUsed(ctx, r.ID), ErrNotFound, "single use")
	_, err = repo.GetLatestPasswordReset(ctx, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteUserCascades(t *testing.T) {
	store := newTestStore(t)
	users, repo := NewUserRepository(store), NewAuthRepository(store)
	ctx := context.Background()
	u, err := users.Create(ctx, newUser("ann@example.com"))
	require.NoError(t, err)
	require.NoError(t, repo.CreateRefreshToken(ctx, u.ID, "h1", time.Now().Add(time.Hour)))
	require.NoError(t, repo.CreatePasswordReset(ctx, u.ID, "c1", time.Now().Add(time.Hour)))

	require.NoError(t, users.Delete(ctx, u.ID))
	for _, table := range []string{"refresh_tokens", "password_resets"} {
		var n int
		require.NoError(t, store.Conn(ctx).QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n))
		assert.Equal(t, 0, n, table)
	}
}
