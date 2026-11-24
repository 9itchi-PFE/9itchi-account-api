package handlers

import (
	"net/http"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
)

// GetUserNote: GET /admin/users/:id/note (admin) the internal note about a user.
func GetUserNote(svc services.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.GetNote(c.Request.Context(), c.Param("id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// SetUserNote: PUT /admin/users/:id/note (admin) replace the internal note about a user.
func SetUserNote(svc services.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.SetUserNoteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.SetNote(c.Request.Context(), c.GetString("user_id"), c.Param("id"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
