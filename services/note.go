package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/repositories"
	"9itchi-account-api/utils"

	"go.uber.org/zap"
)

const maxNoteLength = 2000

// NoteService is the internal note of the team about a student. It is read and written by admins only and never
// returned to the student.
type NoteService interface {
	// GET /admin/users/:id/note: the note ("" when nothing was written)
	GetNote(ctx context.Context, id string) (*api.UserNoteResponse, *models.ApiError)
	// PUT /admin/users/:id/note: replace the note ("" empties it)
	SetNote(ctx context.Context, adminID, id string, req api.SetUserNoteRequest) (*api.UserNoteResponse, *models.ApiError)
}

type noteService struct {
	notes repositories.NoteRepository
	users repositories.UserRepository
}

func NewNoteService(notes repositories.NoteRepository, users repositories.UserRepository) NoteService {
	return &noteService{notes: notes, users: users}
}

func (s *noteService) GetNote(ctx context.Context, id string) (*api.UserNoteResponse, *models.ApiError) {
	if !isUUID(id) {
		return nil, badRequest("invalid user id")
	}
	id = strings.ToLower(id)
	n, err := s.notes.Get(ctx, id)
	if err == nil {
		at := n.UpdatedAt
		return &api.UserNoteResponse{UserID: id, Note: n.Note, UpdatedAt: &at}, nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return nil, noteError(err)
	}
	// No note: say so for a user who exists, 404 for one who does not.
	if _, err := s.users.GetByID(ctx, id); err != nil {
		return nil, noteError(err)
	}
	return &api.UserNoteResponse{UserID: id}, nil
}

func (s *noteService) SetNote(ctx context.Context, adminID, id string, req api.SetUserNoteRequest) (*api.UserNoteResponse, *models.ApiError) {
	if !isUUID(id) {
		return nil, badRequest("invalid user id")
	}
	note := strings.TrimSpace(req.Note)
	if utf8.RuneCountInString(note) > maxNoteLength {
		return nil, badRequest("note must be at most 2000 characters")
	}
	n, err := s.notes.Set(ctx, strings.ToLower(id), note, adminID)
	if err != nil {
		return nil, noteError(err)
	}
	at := n.UpdatedAt
	return &api.UserNoteResponse{UserID: n.UserID, Note: n.Note, UpdatedAt: &at}, nil
}

// noteError never says what was in the note.
func noteError(err error) *models.ApiError {
	if errors.Is(err, repositories.ErrNotFound) {
		return apiError(http.StatusNotFound, "user not found")
	}
	utils.Logger.Error("user note", zap.Error(err))
	return apiError(http.StatusInternalServerError, "internal server error")
}
