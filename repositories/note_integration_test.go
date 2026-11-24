//go:build integration

package repositories

import (
	"context"
	"testing"

	"9itchi-account-api/models/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserNoteIsKeptPerUserAndLeavesWithTheAccount(t *testing.T) {
	store := newTestStore(t)
	users, notes := NewUserRepository(store), NewNoteRepository(store)
	ctx := context.Background()
	const admin = "00000000-0000-4000-8000-000000000001"

	u, err := users.Create(ctx, &db.User{Name: "Ines", Email: "ines.note@example.com", PasswordHash: "x", Role: db.RoleCandidate, Status: db.StatusActive})
	require.NoError(t, err)

	_, err = notes.Get(ctx, u.ID)
	assert.ErrorIs(t, err, ErrNotFound, "nothing written yet")

	n, err := notes.Set(ctx, u.ID, "A appelé pour son reçu.", admin)
	require.NoError(t, err)
	assert.Equal(t, "A appelé pour son reçu.", n.Note)
	assert.Equal(t, admin, n.UpdatedBy)
	n, err = notes.Set(ctx, u.ID, "Réglé.", admin)
	require.NoError(t, err)
	assert.Equal(t, "Réglé.", n.Note, "one note per user: the new one replaces the old one")
	got, err := notes.Get(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Réglé.", got.Note)

	// No note about someone who does not exist.
	_, err = notes.Set(ctx, "99999999-9999-4999-8999-999999999999", "x", admin)
	assert.ErrorIs(t, err, ErrNotFound)

	// The note leaves with the account.
	require.NoError(t, users.Delete(ctx, u.ID))
	_, err = notes.Get(ctx, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}
