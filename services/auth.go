package services

import (
	"context"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/repositories"
)

// AuthService holds all business logic for auth. Handlers stay thin.
type AuthService interface {
	// POST /auth/signup: create candidate account
	Signup(ctx context.Context, req api.SignupRequest) (any, *models.ApiError)
	// POST /auth/login: email + password login, returns encrypted JWT
	Login(ctx context.Context, req api.LoginRequest) (any, *models.ApiError)
	// POST /auth/refresh: rotate access token
	RefreshToken(ctx context.Context, req api.RefreshTokenRequest) (any, *models.ApiError)
	// POST /auth/logout: invalidate refresh token
	Logout(ctx context.Context, userID string) (any, *models.ApiError)
	// POST /auth/forgot-password: send reset code
	ForgotPassword(ctx context.Context, req api.ForgotPasswordRequest) (any, *models.ApiError)
	// POST /auth/reset-password: apply reset code
	ResetPassword(ctx context.Context, req api.ResetPasswordRequest) (any, *models.ApiError)
	// GET /auth/google: start Google OAuth2 sign-in
	GoogleSignIn(ctx context.Context) (any, *models.ApiError)
	// GET /auth/google/callback: finish Google OAuth2 sign-in
	GoogleCallback(ctx context.Context) (any, *models.ApiError)
}

type authService struct {
	repo repositories.AuthRepository
}

func NewAuthService(repo repositories.AuthRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Signup(_ context.Context, _ api.SignupRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) Login(_ context.Context, _ api.LoginRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) RefreshToken(_ context.Context, _ api.RefreshTokenRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) Logout(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) ForgotPassword(_ context.Context, _ api.ForgotPasswordRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) ResetPassword(_ context.Context, _ api.ResetPasswordRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) GoogleSignIn(_ context.Context) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *authService) GoogleCallback(_ context.Context) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}
