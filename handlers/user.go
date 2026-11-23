package handlers

import (
	"net/http"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
)

// GetMe: GET /users/me (user) current profile.
func GetMe(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.GetMe(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// UpdateMe: PUT /users/me (user) update profile.
func UpdateMe(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.UpdateMeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.UpdateMe(c.Request.Context(), c.GetString("user_id"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// DeleteMe: DELETE /users/me (user) delete account and data (right to erasure).
func DeleteMe(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.DeleteMe(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// ListUsers: GET /admin/users (admin) list and search users.
func ListUsers(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.ListUsersRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.ListUsers(c.Request.Context(), c.GetString("user_id"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// GetUser: GET /admin/users/:id (admin) user detail.
func GetUser(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.GetUser(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// SuspendUser: PUT /admin/users/:id/suspend (admin) suspend or reactivate.
func SuspendUser(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.SuspendUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.SuspendUser(c.Request.Context(), c.GetString("user_id"), c.Param("id"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// DeleteUser: DELETE /admin/users/:id (admin) delete user.
func DeleteUser(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.DeleteUser(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// InternalGetUser: GET /internal/users/:id (internal) user lookup for other services.
func InternalGetUser(svc services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.InternalGetUser(c.Request.Context(), c.Param("id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
