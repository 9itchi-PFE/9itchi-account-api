//go:build integration

package repositories

import (
	"context"
	"fmt"
	"testing"

	"9itchi-account-api/models/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUser(email string) *db.User {
	return &db.User{Name: "N " + email, Email: email, PasswordHash: "h", Role: db.RoleCandidate, Status: db.StatusActive}
}

func TestUserCRUD(t *testing.T) {
	store := newTestStore(t)
	repo := NewUserRepository(store)
	ctx := context.Background()

	u, err := repo.Create(ctx, newUser("ann@example.com"))
	require.NoError(t, err)
	assert.Len(t, u.ID, 36)

	_, err = repo.Create(ctx, newUser("ann@example.com"))
	assert.ErrorIs(t, err, ErrConflict, "duplicate email")
	_, err = repo.Create(ctx, newUser("Ann@Example.com"))
	assert.Error(t, err, "mixed-case email rejected by CHECK")
	bad := newUser("x@example.com")
	bad.Role = "root"
	_, err = repo.Create(ctx, bad)
	assert.Error(t, err, "role CHECK")
	bad = newUser("y@example.com")
	bad.Status = "banned"
	_, err = repo.Create(ctx, bad)
	assert.Error(t, err, "status CHECK")

	got, err := repo.GetByEmail(ctx, "ann@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	_, err = repo.GetByEmail(ctx, "none@example.com")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = repo.GetByID(ctx, "00000000-0000-4000-8000-000000000000")
	assert.ErrorIs(t, err, ErrNotFound)

	named, err := repo.UpdateName(ctx, u.ID, "Renamed")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", named.Name)
	assert.True(t, named.UpdatedAt.After(u.UpdatedAt) || named.UpdatedAt.Equal(u.UpdatedAt))
	susp, err := repo.UpdateStatus(ctx, u.ID, db.StatusSuspended)
	require.NoError(t, err)
	assert.Equal(t, db.StatusSuspended, susp.Status)
	require.NoError(t, repo.UpdatePassword(ctx, u.ID, "newhash"))
	got, _ = repo.GetByID(ctx, u.ID)
	assert.Equal(t, "newhash", got.PasswordHash)
	assert.ErrorIs(t, repo.UpdatePassword(ctx, "00000000-0000-4000-8000-000000000000", "x"), ErrNotFound)

	require.NoError(t, repo.Delete(ctx, u.ID))
	assert.ErrorIs(t, repo.Delete(ctx, u.ID), ErrNotFound)
}

func TestAdminExists(t *testing.T) {
	store := newTestStore(t)
	repo := NewUserRepository(store)
	ctx := context.Background()
	ok, err := repo.AdminExists(ctx)
	require.NoError(t, err)
	assert.False(t, ok)
	a := newUser("root@example.com")
	a.Role = db.RoleAdmin
	_, err = repo.Create(ctx, a)
	require.NoError(t, err)
	ok, err = repo.AdminExists(ctx)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestUserListFilters(t *testing.T) {
	store := newTestStore(t)
	repo := NewUserRepository(store)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := repo.Create(ctx, newUser(fmt.Sprintf("user%d@example.com", i)))
		require.NoError(t, err)
	}
	adm := newUser("boss@corp.io")
	adm.Role, adm.Name = db.RoleAdmin, "Big Boss"
	_, err := repo.Create(ctx, adm)
	require.NoError(t, err)
	sus := newUser("gone@example.com")
	sus.Status = db.StatusSuspended
	_, err = repo.Create(ctx, sus)
	require.NoError(t, err)

	all, total, err := repo.List(ctx, UserFilter{Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, 7, total)
	assert.Len(t, all, 7)

	page, total, err := repo.List(ctx, UserFilter{Limit: 3, Offset: 6})
	require.NoError(t, err)
	assert.Equal(t, 7, total, "total ignores paging")
	assert.Len(t, page, 1)

	byName, total, err := repo.List(ctx, UserFilter{Q: "BIG bo", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "boss@corp.io", byName[0].Email)

	byEmail, _, err := repo.List(ctx, UserFilter{Q: "CORP.io", Limit: 10})
	require.NoError(t, err)
	assert.Len(t, byEmail, 1)

	wild, total, err := repo.List(ctx, UserFilter{Q: "%", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total, "LIKE wildcards are literal")
	assert.Empty(t, wild)

	admins, total, err := repo.List(ctx, UserFilter{Role: db.RoleAdmin, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, admins, 1)

	susp, total, err := repo.List(ctx, UserFilter{Status: db.StatusSuspended, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "gone@example.com", susp[0].Email)
}
