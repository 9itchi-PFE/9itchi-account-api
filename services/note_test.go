package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
	"9itchi-account-api/repositories"
	"9itchi-account-api/services/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memNotes keeps notes in memory for the users it knows.
type memNotes struct {
	users map[string]bool
	notes map[string]*db.UserNote
	err   error
}

func (m *memNotes) Get(_ context.Context, userID string) (*db.UserNote, error) {
	if m.err != nil {
		return nil, m.err
	}
	if n, ok := m.notes[userID]; ok {
		return n, nil
	}
	return nil, repositories.ErrNotFound
}

func (m *memNotes) Set(_ context.Context, userID, note, by string) (*db.UserNote, error) {
	if m.err != nil {
		return nil, m.err
	}
	if !m.users[userID] {
		return nil, repositories.ErrNotFound
	}
	n := &db.UserNote{UserID: userID, Note: note, UpdatedBy: by, UpdatedAt: time.Unix(100, 0)}
	m.notes[userID] = n
	return n, nil
}

func TestUserNote(t *testing.T) {
	const student, admin, ghost = "7c9e6679-7425-40de-944b-e07fc1f90ae7", "00000000-0000-4000-8000-000000000001", "99999999-9999-4999-8999-999999999999"
	notes := &memNotes{users: map[string]bool{student: true}, notes: map[string]*db.UserNote{}}
	users := &mocks.UserRepo{Users: map[string]*db.User{student: {ID: student, Name: "Ines"}}}
	svc := NewNoteService(notes, users)
	ctx := context.Background()

	// Nothing written yet: an empty note for a student who exists, 404 for one who does not.
	res, apiErr := svc.GetNote(ctx, student)
	require.Nil(t, apiErr)
	assert.Empty(t, res.Note)
	assert.Nil(t, res.UpdatedAt)
	_, apiErr = svc.GetNote(ctx, ghost)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	res, apiErr = svc.SetNote(ctx, admin, strings.ToUpper(student), api.SetUserNoteRequest{Note: "  A appelé pour son reçu.  "})
	require.Nil(t, apiErr)
	assert.Equal(t, "A appelé pour son reçu.", res.Note)
	assert.Equal(t, admin, notes.notes[student].UpdatedBy, "who wrote it is kept")
	res, apiErr = svc.GetNote(ctx, student)
	require.Nil(t, apiErr)
	assert.Equal(t, "A appelé pour son reçu.", res.Note)
	require.NotNil(t, res.UpdatedAt)

	// An empty note empties it; too long, a bad id and an unknown user are refused.
	res, apiErr = svc.SetNote(ctx, admin, student, api.SetUserNoteRequest{Note: "   "})
	require.Nil(t, apiErr)
	assert.Empty(t, res.Note)
	_, apiErr = svc.SetNote(ctx, admin, student, api.SetUserNoteRequest{Note: strings.Repeat("é", 2001)})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	_, apiErr = svc.SetNote(ctx, admin, student, api.SetUserNoteRequest{Note: strings.Repeat("é", 2000)})
	require.Nil(t, apiErr)
	for _, bad := range []string{"nope", ""} {
		_, apiErr = svc.SetNote(ctx, admin, bad, api.SetUserNoteRequest{Note: "x"})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
		_, apiErr = svc.GetNote(ctx, bad)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	}
	_, apiErr = svc.SetNote(ctx, admin, ghost, api.SetUserNoteRequest{Note: "x"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	// A database error says nothing of the note.
	notes.err = errors.New("db down: A appelé")
	_, apiErr = svc.GetNote(ctx, student)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.EqualError(t, apiErr.Error, "internal server error")
}
