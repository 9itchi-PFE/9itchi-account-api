package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeNotes struct {
	by, id, note string
	err          *models.ApiError
}

func (f *fakeNotes) GetNote(_ context.Context, id string) (*api.UserNoteResponse, *models.ApiError) {
	if f.err != nil {
		return nil, f.err
	}
	return &api.UserNoteResponse{UserID: id, Note: "déjà là"}, nil
}

func (f *fakeNotes) SetNote(_ context.Context, adminID, id string, req api.SetUserNoteRequest) (*api.UserNoteResponse, *models.ApiError) {
	f.by, f.id, f.note = adminID, id, req.Note
	if f.err != nil {
		return nil, f.err
	}
	return &api.UserNoteResponse{UserID: id, Note: req.Note}, nil
}

func noteCall(h gin.HandlerFunc, method, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, "/admin/users/:id/note", func(c *gin.Context) { c.Set("user_id", "admin-1") }, h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/admin/users/u-1/note", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestUserNoteHandlers(t *testing.T) {
	svc := &fakeNotes{}
	w := noteCall(GetUserNote(svc), http.MethodGet, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"note":"déjà là"`)

	w = noteCall(SetUserNote(svc), http.MethodPut, `{"note":"rappeler lundi"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "admin-1", svc.by, "the admin who writes comes from the token")
	assert.Equal(t, "u-1", svc.id)
	assert.Equal(t, "rappeler lundi", svc.note)

	w = noteCall(SetUserNote(svc), http.MethodPut, `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	svc.err = &models.ApiError{Code: http.StatusNotFound, Error: errors.New("user not found")}
	assert.Equal(t, http.StatusNotFound, noteCall(GetUserNote(svc), http.MethodGet, "").Code)
	assert.Equal(t, http.StatusNotFound, noteCall(SetUserNote(svc), http.MethodPut, `{"note":"x"}`).Code)
}
