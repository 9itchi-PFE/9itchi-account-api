package handlers

import (
	"net/http"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
)

// ListSchools: GET /schools (public) the schools offered to students.
func ListSchools(svc services.SchoolService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.List(c.Request.Context())
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// CreateSchool: POST /admin/schools (admin) add a school to the list.
func CreateSchool(svc services.SchoolService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.CreateSchoolRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.Create(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusCreated, result)
	}
}

// UpdateSchool: PUT /admin/schools/:code (admin) rename a school.
func UpdateSchool(svc services.SchoolService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.UpdateSchoolRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.Update(c.Request.Context(), c.Param("code"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// DeleteSchool: DELETE /admin/schools/:code (admin) remove a school from the list.
func DeleteSchool(svc services.SchoolService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiErr := svc.Delete(c.Request.Context(), c.Param("code")); apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
