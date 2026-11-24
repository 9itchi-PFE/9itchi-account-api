package handlers

import (
	"errors"
	"io"
	"net/http"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
)

// Signup: POST /auth/signup (public) create candidate account.
func Signup(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.SignupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.Signup(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusCreated, result)
	}
}

// Login: POST /auth/login (public) email + password login, returns encrypted JWT.
func Login(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.Login(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// RefreshToken: POST /auth/refresh (public) rotate access token.
func RefreshToken(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.RefreshToken(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Logout: POST /auth/logout (user) invalidate refresh token.
func Logout(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.LogoutRequest
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.Logout(c.Request.Context(), c.GetString("user_id"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// ForgotPassword: POST /auth/forgot-password (public) send reset code.
func ForgotPassword(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.ForgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.ForgotPassword(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// ResetPassword: POST /auth/reset-password (public) apply reset code.
func ResetPassword(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.ResetPassword(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

const (
	googleStateCookie = "google_oauth_state"
	googleStateMaxAge = 10 * 60
)

// GoogleSignIn: GET /auth/google (public) start Google OAuth2 sign-in.
func GoogleSignIn(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start, apiErr := svc.GoogleSignIn(c.Request.Context())
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(googleStateCookie, start.State, googleStateMaxAge, "/auth/google", "", isHTTPS(c), true)
		c.Redirect(http.StatusFound, start.URL)
	}
}

// GoogleCallback: GET /auth/google/callback (public) finish Google OAuth2 sign-in.
func GoogleCallback(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieState, _ := c.Cookie(googleStateCookie)
		result, apiErr := svc.GoogleCallback(c.Request.Context(), api.GoogleCallbackRequest{
			Code:        c.Query("code"),
			State:       c.Query("state"),
			CookieState: cookieState,
		})
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(googleStateCookie, "", -1, "/auth/google", "", isHTTPS(c), true)
		c.Redirect(http.StatusFound, result.URL)
	}
}

func isHTTPS(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}
