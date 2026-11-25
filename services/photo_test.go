package services

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"9itchi-account-api/repositories"
)

type memPhotos struct {
	photos map[string][]byte
	known  map[string]bool
	err    error
}

func (m *memPhotos) Set(_ context.Context, id string, data []byte) error {
	if m.err != nil {
		return m.err
	}
	if !m.known[id] {
		return repositories.ErrNotFound
	}
	m.photos[id] = data
	return nil
}

func (m *memPhotos) Get(_ context.Context, id string) ([]byte, error) {
	if d, ok := m.photos[id]; ok {
		return d, nil
	}
	return nil, repositories.ErrNotFound
}

func (m *memPhotos) Delete(_ context.Context, id string) error {
	delete(m.photos, id)
	return nil
}

func encoded(t *testing.T, w, h int, asPNG bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Red on the left half, transparent on the right (a PNG must not turn black).
			if x < w/2 {
				img.Set(x, y, color.NRGBA{R: 200, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	if asPNG {
		require.NoError(t, png.Encode(&buf, img))
	} else {
		require.NoError(t, jpeg.Encode(&buf, img, nil))
	}
	return buf.Bytes()
}

func TestProfilePhotoIsStoredSquareAndSmall(t *testing.T) {
	repo := &memPhotos{photos: map[string][]byte{}, known: map[string]bool{userID: true}}
	svc := NewPhotoService(repo)

	require.Nil(t, svc.Upload(context.Background(), userID, bytes.NewReader(encoded(t, 1200, 800, true))))
	got, apiErr := svc.Get(context.Background(), userID)
	require.Nil(t, apiErr)
	img, format, err := image.Decode(bytes.NewReader(got))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
	assert.Equal(t, image.Rect(0, 0, 400, 400), img.Bounds(), "a centred square, reduced")
	r, g, b, _ := img.At(390, 200).RGBA()
	assert.Greater(t, r>>8, uint32(240), "transparent becomes white, not black")
	assert.Greater(t, g>>8, uint32(240))
	assert.Greater(t, b>>8, uint32(240))
	assert.Less(t, len(got), 100_000)

	require.Nil(t, svc.Upload(context.Background(), userID, bytes.NewReader(encoded(t, 120, 300, false))))
	got, _ = svc.Get(context.Background(), userID)
	small, _, err := image.Decode(bytes.NewReader(got))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 120, 120), small.Bounds(), "a small photo is not enlarged")

	require.Nil(t, svc.Delete(context.Background(), userID))
	_, apiErr = svc.Get(context.Background(), userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
	require.Nil(t, svc.Delete(context.Background(), userID), "removing none is fine")
}

func TestProfilePhotoRefusals(t *testing.T) {
	repo := &memPhotos{photos: map[string][]byte{}, known: map[string]bool{userID: true}}
	svc := NewPhotoService(repo)
	for name, data := range map[string][]byte{
		"a PDF":              []byte("%PDF-1.4 ..."),
		"a broken JPEG":      {0xFF, 0xD8, 0xFF, 0x00, 0x01},
		"text named as .png": []byte("hello"),
	} {
		apiErr := svc.Upload(context.Background(), userID, bytes.NewReader(data))
		require.NotNil(t, apiErr, name)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code, name)
	}
	apiErr := svc.Upload(context.Background(), userID, bytes.NewReader(make([]byte, MaxPhotoBytes+1)))
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusRequestEntityTooLarge, apiErr.Code)

	apiErr = svc.Upload(context.Background(), "33333333-3333-4333-8333-333333333333", bytes.NewReader(encoded(t, 10, 10, true)))
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	repo.err = errors.New("db down")
	apiErr = svc.Upload(context.Background(), userID, bytes.NewReader(encoded(t, 10, 10, true)))
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Empty(t, repo.photos)
}
