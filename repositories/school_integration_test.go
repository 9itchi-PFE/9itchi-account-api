//go:build integration

package repositories

import (
	"context"
	"testing"

	"9itchi-account-api/models/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchoolsAreSeededAndEditable(t *testing.T) {
	store := newTestStore(t)
	schools := NewSchoolRepository(store)
	ctx := context.Background()

	list, err := schools.List(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, list, "the migration seeds the schools students could already pick")
	assert.Equal(t, "ENIS", list[0].Code)
	assert.Equal(t, "ENETCOM", list[1].Code)

	created, err := schools.Create(ctx, &db.School{Code: "TESTSCH", NameFR: "fr", NameEN: "en", NameAR: "ar"})
	require.NoError(t, err)
	after, err := schools.List(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(list)+1)
	assert.Equal(t, created.Code, after[len(after)-1].Code, "a new school goes last")

	_, err = schools.Create(ctx, &db.School{Code: "TESTSCH", NameFR: "x", NameEN: "x", NameAR: "x"})
	assert.ErrorIs(t, err, ErrConflict)
	_, err = schools.Create(ctx, &db.School{Code: "bad code", NameFR: "x", NameEN: "x", NameAR: "x"})
	assert.Error(t, err, "the database refuses a malformed code")

	updated, err := schools.Update(ctx, &db.School{Code: "TESTSCH", NameFR: "fr2", NameEN: "en2", NameAR: "ar2"})
	require.NoError(t, err)
	assert.Equal(t, "en2", updated.NameEN)
	assert.Equal(t, created.Position, updated.Position, "renaming keeps the place in the list")
	_, err = schools.Update(ctx, &db.School{Code: "NOPE", NameFR: "x", NameEN: "x", NameAR: "x"})
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, schools.Delete(ctx, "TESTSCH"))
	assert.ErrorIs(t, schools.Delete(ctx, "TESTSCH"), ErrNotFound)
}
