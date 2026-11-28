package handlers

import (
	"context"
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

// VerifyEmail: POST /auth/verify-email (public) check the emailed code and sign the user in.
func VerifyEmail(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.VerifyEmail(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// ResendVerification: POST /auth/resend-verification (public) email a new code; always a generic answer.
func ResendVerification(svc services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.ResendVerificationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.ResendVerification(c.Request.Context(), req)
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
	googleStateCookie   = "google_oauth_state"
	linkedinStateCookie = "linkedin_oauth_state"
	oauthStateMaxAge    = 10 * 60
	// The service sits behind a gateway prefix (/api/account), so a provider-specific path would never match the browser URL.
	oauthCookiePath = "/"
)

// GoogleSignIn: GET /auth/google (public) start Google OAuth2 sign-in.
func GoogleSignIn(svc services.AuthService) gin.HandlerFunc {
	return oauthSignIn(svc.GoogleSignIn, googleStateCookie, oauthCookiePath)
}

// GoogleCallback: GET /auth/google/callback (public) finish Google OAuth2 sign-in.
func GoogleCallback(svc services.AuthService) gin.HandlerFunc {
	return oauthCallback(svc.GoogleCallback, googleStateCookie, oauthCookiePath)
}

// LinkedInSignIn: GET /auth/linkedin (public) start LinkedIn OpenID Connect sign-in.
func LinkedInSignIn(svc services.AuthService) gin.HandlerFunc {
	return oauthSignIn(svc.LinkedInSignIn, linkedinStateCookie, oauthCookiePath)
}

// LinkedInCallback: GET /auth/linkedin/callback (public) finish LinkedIn sign-in.
func LinkedInCallback(svc services.AuthService) gin.HandlerFunc {
	return oauthCallback(svc.LinkedInCallback, linkedinStateCookie, oauthCookiePath)
}

// oauthSignIn stores the state in an HttpOnly cookie scoped to the provider path and redirects to the consent page.
func oauthSignIn(start func(context.Context) (*api.OAuthStart, *models.ApiError), cookie, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, apiErr := start(c.Request.Context())
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(cookie, res.State, oauthStateMaxAge, path, "", isHTTPS(c), true)
		c.Redirect(http.StatusFound, res.URL)
	}
}

// oauthCallback hands the query values and the state cookie to the service, clears the cookie and follows its redirect.
func oauthCallback(finish func(context.Context, api.OAuthCallbackRequest) (*api.OAuthRedirect, *models.ApiError), cookie, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieState, _ := c.Cookie(cookie)
		res, apiErr := finish(c.Request.Context(), api.OAuthCallbackRequest{
			Code:        c.Query("code"),
			State:       c.Query("state"),
			CookieState: cookieState,
		})
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(cookie, "", -1, path, "", isHTTPS(c), true)
		c.Redirect(http.StatusFound, res.URL)
	}
}

func isHTTPS(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}
