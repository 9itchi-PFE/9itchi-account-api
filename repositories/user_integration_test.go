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

	assert.Equal(t, "fr", u.Language, "French when none is given")
	arabic := newUser("arabic@example.com")
	arabic.Language = "ar"
	ar, err := repo.Create(ctx, arabic)
	require.NoError(t, err)
	assert.Equal(t, "ar", ar.Language)
	en, err := repo.UpdateLanguage(ctx, u.ID, "en")
	require.NoError(t, err)
	assert.Equal(t, "en", en.Language)
	_, err = repo.UpdateLanguage(ctx, u.ID, "de")
	assert.Error(t, err, "CHECK language")
	_, err = repo.UpdateLanguage(ctx, "00000000-0000-0000-0000-000000000000", "en")
	assert.ErrorIs(t, err, ErrNotFound)

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

	enis := "ENIS"
	withSchool, err := repo.UpdateProfile(ctx, u.ID, "Ann", ProfilePatch{School: &enis})
	require.NoError(t, err)
	require.NotNil(t, withSchool.School)
	assert.Equal(t, "ENIS", *withSchool.School)
	kept, err := repo.UpdateProfile(ctx, u.ID, "Ann", ProfilePatch{})
	require.NoError(t, err)
	require.NotNil(t, kept.School, "nil keeps the school")
	empty := ""
	cleared, err := repo.UpdateProfile(ctx, u.ID, "Ann", ProfilePatch{School: &empty})
	require.NoError(t, err)
	assert.Nil(t, cleared.School)

	// The rest of the profile: each field set on its own, kept when left out, cleared by "".
	phone, city, spec, year := "+21620000000", "Sfax", "Génie logiciel", "ing3"
	li, gh := "https://www.linkedin.com/in/ann", "https://github.com/ann"
	full, err := repo.UpdateProfile(ctx, u.ID, "Ann", ProfilePatch{Phone: &phone, City: &city, Speciality: &spec, Year: &year, LinkedIn: &li, GitHub: &gh})
	require.NoError(t, err)
	require.NotNil(t, full.GitHub)
	assert.Equal(t, []string{phone, city, spec, year, li, gh}, []string{*full.Phone, *full.City, *full.Speciality, *full.Year, *full.LinkedIn, *full.GitHub})
	part, err := repo.UpdateProfile(ctx, u.ID, "Ann", ProfilePatch{City: &empty})
	require.NoError(t, err)
	assert.Nil(t, part.City)
	require.NotNil(t, part.Phone, "what is left out stays")
	assert.Equal(t, phone, *part.Phone)
	unknown := "phd"
	_, err = repo.UpdateProfile(ctx, u.ID, "Ann", ProfilePatch{Year: &unknown})
	assert.Error(t, err, "the table refuses a year it does not know")

	// The photo: kept apart, the user view only says there is one; list queries never read it.
	photos := NewPhotoRepository(store)
	_, err = photos.Get(ctx, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, photos.Set(ctx, u.ID, []byte{0xFF, 0xD8, 0xFF, 1, 2, 3}))
	withPhoto, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, withPhoto.HasPhoto)
	data, err := photos.Get(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xFF, 0xD8, 0xFF, 1, 2, 3}, data)
	listed, _, err := repo.List(ctx, UserFilter{Limit: 50})
	require.NoError(t, err)
	found := false
	for _, l := range listed {
		if l.ID == u.ID {
			found = true
			assert.True(t, l.HasPhoto)
		}
	}
	assert.True(t, found)
	require.NoError(t, photos.Delete(ctx, u.ID))
	noPhoto, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.False(t, noPhoto.HasPhoto)
	assert.ErrorIs(t, photos.Set(ctx, "00000000-0000-4000-8000-000000000000", []byte{1}), ErrNotFound)

	named, err := repo.UpdateName(ctx, u.ID, "Renamed")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", named.Name)
	assert.True(t, named.UpdatedAt.After(u.UpdatedAt) || named.UpdatedAt.Equal(u.UpdatedAt))
	susp, err := repo.UpdateStatus(ctx, u.ID, db.StatusSuspended)
	require.NoError(t, err)
	assert.Equal(t, db.StatusSuspended, susp.Status)
	assert.False(t, susp.EmailVerified, "new accounts start unverified")
	verified, err := repo.MarkEmailVerified(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, verified.EmailVerified)
	_, err = repo.MarkEmailVerified(ctx, "00000000-0000-4000-8000-000000000000")
	assert.ErrorIs(t, err, ErrNotFound)
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
