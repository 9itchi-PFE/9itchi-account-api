package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"9itchi-account-api/models"
)

type fakePhotos struct {
	got    []byte
	userID string
	data   []byte
}

func (f *fakePhotos) Upload(_ context.Context, userID string, r io.Reader) *models.ApiError {
	f.userID = userID
	f.got, _ = io.ReadAll(r)
	if bytes.HasPrefix(f.got, []byte("bad")) {
		return &models.ApiError{Code: http.StatusBadRequest, Error: errors.New("the photo must be a JPEG or PNG image")}
	}
	return nil
}

func (f *fakePhotos) Get(_ context.Context, _ string) ([]byte, *models.ApiError) {
	if f.data == nil {
		return nil, &models.ApiError{Code: http.StatusNotFound, Error: errors.New("no profile photo")}
	}
	return f.data, nil
}

func (f *fakePhotos) Delete(_ context.Context, userID string) *models.ApiError {
	f.userID = userID
	f.data = nil
	return nil
}

func photoRequest(t *testing.T, h gin.HandlerFunc, field string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile(field, "me.jpg")
	require.NoError(t, err)
	_, _ = part.Write(content)
	require.NoError(t, mw.Close())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/users/me/photo", &body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Set("user_id", "uid-1")
	h(c)
	return w
}

func TestPhotoHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakePhotos{}

	w := photoRequest(t, UploadPhoto(svc), "file", []byte{0xFF, 0xD8, 0xFF, 1})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"has_photo":true}`, w.Body.String())
	assert.Equal(t, "uid-1", svc.userID, "the photo goes to the caller")
	assert.Equal(t, []byte{0xFF, 0xD8, 0xFF, 1}, svc.got)

	w = photoRequest(t, UploadPhoto(svc), "photo", []byte{1})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"multipart field 'file' is required","status":400}`, w.Body.String())

	w = photoRequest(t, UploadPhoto(svc), "file", []byte("bad bytes"))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = photoRequest(t, UploadPhoto(svc), "file", make([]byte, 5<<20+128<<10))
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	w = do(GetPhoto(svc), http.MethodGet, "/users/me/photo", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	svc.data = []byte{0xFF, 0xD8, 0xFF, 9}
	w = do(GetPhoto(svc), http.MethodGet, "/users/me/photo", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, []byte{0xFF, 0xD8, 0xFF, 9}, w.Body.Bytes())

	w = do(DeletePhoto(svc), http.MethodDelete, "/users/me/photo", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"has_photo":false}`, w.Body.String())
	assert.Nil(t, svc.data)
}
