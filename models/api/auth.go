package api

// SignupRequest is the validated body of Signup.
type SignupRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest is the validated body of Login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

// RefreshTokenRequest is the validated body of RefreshToken.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest is the optional body of Logout; without refresh_token every session of the user is revoked.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ForgotPasswordRequest is the validated body of ForgotPassword.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

// ResetPasswordRequest is the validated body of ResetPassword.
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email,max=254"`
	Code        string `json:"code" binding:"required,len=6,numeric"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// AuthResponse is returned by signup and login.
type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// TokenResponse is returned by refresh.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// GoogleStart is what the handler needs to begin Google sign-in: the consent URL and the state to store in the cookie.
type GoogleStart struct {
	URL   string
	State string
}

// GoogleCallbackRequest carries the callback query values and the state cookie.
type GoogleCallbackRequest struct {
	Code        string
	State       string
	CookieState string
}

// GoogleRedirect is the browser redirect target after the callback (success or failure).
type GoogleRedirect struct {
	URL string
}

// MessageResponse is a plain acknowledgement body.
type MessageResponse struct {
	Message string `json:"message"`
}
