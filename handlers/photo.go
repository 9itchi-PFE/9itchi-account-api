package handlers

import (
	"errors"
	"net/http"

	"9itchi-account-api/models"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
)

// multipartOverhead is room for the multipart headers around the photo.
const multipartOverhead = 64 << 10

// UploadPhoto: PUT /users/me/photo (user) multipart field "file", JPEG or PNG, 5 MB max; stored square and reduced.
func UploadPhoto(svc services.PhotoService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxPhotoBytes+multipartOverhead)
		header, err := c.FormFile("file")
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				models.FormatError(c, &models.ApiError{Code: http.StatusRequestEntityTooLarge, Error: errors.New("photo too large (max 5 MB)")})
				return
			}
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: errors.New("multipart field 'file' is required")})
			return
		}
		f, err := header.Open()
		if err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: errors.New("could not read file")})
			return
		}
		defer f.Close()
		if apiErr := svc.Upload(c.Request.Context(), c.GetString("user_id"), f); apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, gin.H{"has_photo": true})
	}
}

// GetPhoto: GET /users/me/photo (user) the profile photo, JPEG.
func GetPhoto(svc services.PhotoService) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, apiErr := svc.Get(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.Header("Cache-Control", "private, no-store")
		c.Data(http.StatusOK, "image/jpeg", data)
	}
}

// DeletePhoto: DELETE /users/me/photo (user) removes the profile photo.
func DeletePhoto(svc services.PhotoService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiErr := svc.Delete(c.Request.Context(), c.GetString("user_id")); apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, gin.H{"has_photo": false})
	}
}
