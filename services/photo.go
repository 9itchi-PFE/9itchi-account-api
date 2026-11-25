package services

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // PNG photos are accepted
	"io"
	"net/http"

	"9itchi-account-api/models"
	"9itchi-account-api/repositories"
)

const (
	// MaxPhotoBytes is the largest photo a student can send (5 MB); it is stored reduced, far smaller.
	MaxPhotoBytes = 5 << 20
	// photoSide is the side of the stored square photo, in pixels.
	photoSide = 400
	// maxPhotoPixels refuses images too large to decode safely (about 40 megapixels).
	maxPhotoPixels = 40_000_000
)

// PhotoService keeps the student's profile photo: one square JPEG, cut from the middle of what they send.
type PhotoService interface {
	// PUT /users/me/photo
	Upload(ctx context.Context, userID string, r io.Reader) *models.ApiError
	// GET /users/me/photo
	Get(ctx context.Context, userID string) ([]byte, *models.ApiError)
	// DELETE /users/me/photo
	Delete(ctx context.Context, userID string) *models.ApiError
}

type photoService struct {
	repo repositories.PhotoRepository
}

func NewPhotoService(repo repositories.PhotoRepository) PhotoService {
	return &photoService{repo: repo}
}

func (s *photoService) Upload(ctx context.Context, userID string, r io.Reader) *models.ApiError {
	if !isUUID(userID) {
		return badRequest("invalid user id")
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxPhotoBytes+1))
	if err != nil {
		return badRequest("could not read the photo")
	}
	if len(data) > MaxPhotoBytes {
		return &models.ApiError{Code: http.StatusRequestEntityTooLarge, Error: errors.New("photo too large (max 5 MB)")}
	}
	square, apiErr := squarePhoto(data)
	if apiErr != nil {
		return apiErr
	}
	if err := s.repo.Set(ctx, userID, square); errors.Is(err, repositories.ErrNotFound) {
		return notFound()
	} else if err != nil {
		return internalError("photo save", err)
	}
	return nil
}

func (s *photoService) Get(ctx context.Context, userID string) ([]byte, *models.ApiError) {
	if !isUUID(userID) {
		return nil, badRequest("invalid user id")
	}
	data, err := s.repo.Get(ctx, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, &models.ApiError{Code: http.StatusNotFound, Error: errors.New("no photo")}
	}
	if err != nil {
		return nil, internalError("photo read", err)
	}
	return data, nil
}

func (s *photoService) Delete(ctx context.Context, userID string) *models.ApiError {
	if !isUUID(userID) {
		return badRequest("invalid user id")
	}
	if err := s.repo.Delete(ctx, userID); err != nil {
		return internalError("photo delete", err)
	}
	return nil
}

// squarePhoto reads a JPEG or PNG, cuts the largest centred square, reduces it to photoSide and writes it as JPEG
// on white (a transparent PNG does not turn black). Anything else, or an image too large to decode, is refused.
func squarePhoto(data []byte) ([]byte, *models.ApiError) {
	if !bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) && !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return nil, badRequest("only JPEG or PNG photos are accepted")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, badRequest("the photo cannot be read")
	}
	if cfg.Width*cfg.Height > maxPhotoPixels {
		return nil, badRequest("the photo is too large in pixels")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, badRequest("the photo cannot be read")
	}
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	x0, y0 := b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2
	out := min(side, photoSide)
	scaled := image.NewRGBA(image.Rect(0, 0, out, out))
	for y := 0; y < out; y++ {
		for x := 0; x < out; x++ {
			scaled.Set(x, y, img.At(x0+x*side/out, y0+y*side/out))
		}
	}
	dst := image.NewRGBA(scaled.Bounds())
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), scaled, image.Point{}, draw.Over)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, internalError("photo encode", err)
	}
	return buf.Bytes(), nil
}
