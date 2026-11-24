package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"9itchi-account-api/core"
	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
	"9itchi-account-api/models/mapping"
	"9itchi-account-api/repositories"
	"9itchi-account-api/utils"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTTL              = 15 * time.Minute
	refreshTTL             = 7 * 24 * time.Hour
	resetCodeTTL           = 15 * time.Minute
	maxResetAttempts       = 5
	errInvalidCreds        = "invalid credentials"
	errInvalidRefresh      = "invalid refresh token"
	errInvalidResetMsg     = "invalid or expired code"
	errGoogleUnavailable   = "google sign-in unavailable"
	errLinkedInUnavailable = "linkedin sign-in unavailable"
)

// AuthService holds all business logic for auth. Handlers stay thin.
type AuthService interface {
	// POST /auth/signup: create candidate account
	Signup(ctx context.Context, req api.SignupRequest) (*api.AuthResponse, *models.ApiError)
	// POST /auth/login: email + password login, returns access and refresh tokens
	Login(ctx context.Context, req api.LoginRequest) (*api.AuthResponse, *models.ApiError)
	// POST /auth/refresh: rotate the refresh token and issue a new access token
	RefreshToken(ctx context.Context, req api.RefreshTokenRequest) (*api.TokenResponse, *models.ApiError)
	// POST /auth/logout: revoke the given refresh token (or all of the user's when none is given)
	Logout(ctx context.Context, userID string, req api.LogoutRequest) (*api.MessageResponse, *models.ApiError)
	// POST /auth/forgot-password: send reset code (always a generic answer)
	ForgotPassword(ctx context.Context, req api.ForgotPasswordRequest) (*api.MessageResponse, *models.ApiError)
	// POST /auth/reset-password: apply reset code
	ResetPassword(ctx context.Context, req api.ResetPasswordRequest) (*api.MessageResponse, *models.ApiError)
	// GET /auth/google: start Google OAuth2 sign-in (503 when not configured)
	GoogleSignIn(ctx context.Context) (*api.GoogleStart, *models.ApiError)
	// GET /auth/google/callback: finish Google OAuth2 sign-in; every failure is a redirect to the frontend sign-in page
	GoogleCallback(ctx context.Context, req api.GoogleCallbackRequest) (*api.GoogleRedirect, *models.ApiError)
	// GET /auth/linkedin: start LinkedIn (OpenID Connect) sign-in (503 when not configured)
	LinkedInSignIn(ctx context.Context) (*api.OAuthStart, *models.ApiError)
	// GET /auth/linkedin/callback: finish LinkedIn sign-in; every failure is a redirect to the frontend sign-in page
	LinkedInCallback(ctx context.Context, req api.OAuthCallbackRequest) (*api.OAuthRedirect, *models.ApiError)
	// BootstrapAdmin creates the first admin from ADMIN_EMAIL/ADMIN_PASSWORD when no admin exists.
	BootstrapAdmin(ctx context.Context) error
}

type authService struct {
	cfg      utils.AppConfig
	users    repositories.UserRepository
	repo     repositories.AuthRepository
	tx       repositories.Transactor
	mailer   core.Mailer
	google   core.OAuthProvider
	linkedin core.OAuthProvider
}

func NewAuthService(cfg utils.AppConfig, users repositories.UserRepository, repo repositories.AuthRepository, tx repositories.Transactor, mailer core.Mailer, google, linkedin core.OAuthProvider) AuthService {
	return &authService{cfg: cfg, users: users, repo: repo, tx: tx, mailer: mailer, google: google, linkedin: linkedin}
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// burnPasswordCompare spends the same time as a real check so unknown emails are not distinguishable by latency.
func burnPasswordCompare(password string) {
	dummyHashOnce.Do(func() { dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-Password-1"), bcrypt.DefaultCost) })
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

func (s *authService) issueTokens(ctx context.Context, u *db.User) (access, refresh string, err error) {
	claims := utils.Claims{ID: u.ID, Email: u.Email, Role: u.Role, Type: utils.TokenAccess}
	if access, err = utils.IssueToken(s.cfg, claims, accessTTL); err != nil {
		return "", "", err
	}
	claims.Type = utils.TokenRefresh
	if refresh, err = utils.IssueToken(s.cfg, claims, refreshTTL); err != nil {
		return "", "", err
	}
	if err = s.repo.CreateRefreshToken(ctx, u.ID, hashToken(refresh), time.Now().Add(refreshTTL)); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *authService) Signup(ctx context.Context, req api.SignupRequest) (*api.AuthResponse, *models.ApiError) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, badRequest("name is required")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, badRequest(err.Error())
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, internalError("signup hash", err)
	}
	var school *string
	if v := strings.TrimSpace(req.School); v != "" {
		school = &v
	}
	user, err := s.users.Create(ctx, &db.User{
		Name: name, Email: normalizeEmail(req.Email), PasswordHash: string(hash),
		Role: db.RoleCandidate, Status: db.StatusActive, School: school,
	})
	if errors.Is(err, repositories.ErrConflict) {
		return nil, apiError(http.StatusConflict, "email already registered")
	}
	if err != nil {
		return nil, internalError("signup create", err)
	}
	access, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, internalError("signup tokens", err)
	}
	return &api.AuthResponse{User: mapping.UserToResponse(user), AccessToken: access, RefreshToken: refresh}, nil
}

func (s *authService) Login(ctx context.Context, req api.LoginRequest) (*api.AuthResponse, *models.ApiError) {
	user, err := s.users.GetByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, repositories.ErrNotFound) {
		burnPasswordCompare(req.Password)
		return nil, apiError(http.StatusUnauthorized, errInvalidCreds)
	}
	if err != nil {
		return nil, internalError("login lookup", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, apiError(http.StatusUnauthorized, errInvalidCreds)
	}
	if user.Status == db.StatusSuspended {
		return nil, apiError(http.StatusForbidden, "account suspended")
	}
	access, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, internalError("login tokens", err)
	}
	return &api.AuthResponse{User: mapping.UserToResponse(user), AccessToken: access, RefreshToken: refresh}, nil
}

func (s *authService) RefreshToken(ctx context.Context, req api.RefreshTokenRequest) (*api.TokenResponse, *models.ApiError) {
	claims, err := utils.ParseToken(s.cfg, req.RefreshToken)
	if err != nil || claims.Type != utils.TokenRefresh {
		return nil, apiError(http.StatusUnauthorized, errInvalidRefresh)
	}
	stored, err := s.repo.ConsumeRefreshToken(ctx, hashToken(req.RefreshToken))
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, apiError(http.StatusUnauthorized, errInvalidRefresh)
	}
	if err != nil {
		return nil, internalError("refresh consume", err)
	}
	if subtle.ConstantTimeCompare([]byte(stored.UserID), []byte(claims.ID)) != 1 {
		return nil, apiError(http.StatusUnauthorized, errInvalidRefresh)
	}
	user, err := s.users.GetByID(ctx, stored.UserID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, apiError(http.StatusUnauthorized, errInvalidRefresh)
	}
	if err != nil {
		return nil, internalError("refresh user", err)
	}
	if user.Status == db.StatusSuspended {
		return nil, apiError(http.StatusForbidden, "account suspended")
	}
	access, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, internalError("refresh tokens", err)
	}
	return &api.TokenResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *authService) Logout(ctx context.Context, userID string, req api.LogoutRequest) (*api.MessageResponse, *models.ApiError) {
	if !isUUID(userID) {
		return nil, apiError(http.StatusUnauthorized, "unauthorized")
	}
	var err error
	if req.RefreshToken != "" {
		err = s.repo.RevokeRefreshToken(ctx, userID, hashToken(req.RefreshToken))
	} else {
		err = s.repo.RevokeAllRefreshTokens(ctx, userID)
	}
	if err != nil {
		return nil, internalError("logout", err)
	}
	return &api.MessageResponse{Message: "logged out"}, nil
}

const forgotMessage = "if the account exists, a reset code has been sent"

func (s *authService) ForgotPassword(ctx context.Context, req api.ForgotPasswordRequest) (*api.MessageResponse, *models.ApiError) {
	generic := &api.MessageResponse{Message: forgotMessage}
	email := normalizeEmail(req.Email)
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repositories.ErrNotFound) {
		return generic, nil
	}
	if err != nil {
		utils.Logger.Error("forgot lookup", zap.Error(err))
		return generic, nil
	}
	if user.Status != db.StatusActive {
		return generic, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		utils.Logger.Error("forgot code", zap.Error(err))
		return generic, nil
	}
	code := fmt.Sprintf("%06d", n.Int64())
	if err := s.repo.CreatePasswordReset(ctx, user.ID, hashCode(s.cfg.SecretKey, user.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		utils.Logger.Error("forgot store", zap.Error(err))
		return generic, nil
	}
	if err := s.mailer.SendPasswordResetCode(ctx, user.Email, code); err != nil {
		utils.Logger.Error("forgot mail", zap.Error(err))
	}
	return generic, nil
}

func (s *authService) ResetPassword(ctx context.Context, req api.ResetPasswordRequest) (*api.MessageResponse, *models.ApiError) {
	if err := validatePassword(req.NewPassword); err != nil {
		return nil, badRequest(err.Error())
	}
	invalid := badRequest(errInvalidResetMsg)
	user, err := s.users.GetByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, internalError("reset lookup", err)
	}
	reset, err := s.repo.GetLatestPasswordReset(ctx, user.ID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, internalError("reset load", err)
	}
	if reset.Attempts >= maxResetAttempts || !time.Now().Before(reset.ExpiresAt) {
		return nil, invalid
	}
	if subtle.ConstantTimeCompare([]byte(reset.CodeHash), []byte(hashCode(s.cfg.SecretKey, user.ID, req.Code))) != 1 {
		if _, err := s.repo.RecordResetAttempt(ctx, reset.ID); err != nil {
			return nil, internalError("reset attempt", err)
		}
		return nil, invalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, internalError("reset hash", err)
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.MarkPasswordResetUsed(ctx, reset.ID); err != nil {
			return err
		}
		if err := s.users.UpdatePassword(ctx, user.ID, string(hash)); err != nil {
			return err
		}
		return s.repo.RevokeAllRefreshTokens(ctx, user.ID)
	})
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, internalError("reset apply", err)
	}
	return &api.MessageResponse{Message: "password updated"}, nil
}

func (s *authService) GoogleSignIn(_ context.Context) (*api.GoogleStart, *models.ApiError) {
	return oauthStart(s.google, errGoogleUnavailable)
}

func (s *authService) GoogleCallback(ctx context.Context, req api.GoogleCallbackRequest) (*api.GoogleRedirect, *models.ApiError) {
	return s.oauthCallback(ctx, s.google, "google", errGoogleUnavailable, req)
}

func (s *authService) LinkedInSignIn(_ context.Context) (*api.OAuthStart, *models.ApiError) {
	return oauthStart(s.linkedin, errLinkedInUnavailable)
}

func (s *authService) LinkedInCallback(ctx context.Context, req api.OAuthCallbackRequest) (*api.OAuthRedirect, *models.ApiError) {
	return s.oauthCallback(ctx, s.linkedin, "linkedin", errLinkedInUnavailable, req)
}

// oauthStart returns the provider consent URL with a fresh random state (503 when the provider is not configured).
func oauthStart(provider core.OAuthProvider, unavailable string) (*api.OAuthStart, *models.ApiError) {
	if provider == nil {
		return nil, apiError(http.StatusServiceUnavailable, unavailable)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, internalError("oauth state", err)
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	return &api.OAuthStart{URL: provider.AuthURL(state), State: state}, nil
}

// oauthCallback checks the state, exchanges the code and signs the user in (creating a password-less account if needed).
// Every failure is a redirect to the frontend sign-in page with error=<name>_failed; the reason is only logged.
func (s *authService) oauthCallback(ctx context.Context, provider core.OAuthProvider, name, unavailable string, req api.OAuthCallbackRequest) (*api.OAuthRedirect, *models.ApiError) {
	if provider == nil {
		return nil, apiError(http.StatusServiceUnavailable, unavailable)
	}
	frontend := strings.TrimRight(s.cfg.FrontendURL, "/")
	failed := &api.OAuthRedirect{URL: frontend + "/auth/signin?error=" + name + "_failed"}

	if req.CookieState == "" || req.Code == "" || subtle.ConstantTimeCompare([]byte(req.State), []byte(req.CookieState)) != 1 {
		return failed, nil
	}
	profile, err := provider.Exchange(ctx, req.Code)
	if err != nil {
		utils.Logger.Warn("oauth exchange", zap.String("provider", name), zap.Error(err))
		return failed, nil
	}
	email := normalizeEmail(profile.Email)
	if email == "" || !profile.EmailVerified {
		return failed, nil
	}
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repositories.ErrNotFound) {
		user, err = s.createOAuthUser(ctx, email, profile.Name)
	}
	if err != nil {
		utils.Logger.Error("oauth user", zap.String("provider", name), zap.Error(err))
		return failed, nil
	}
	if user.Status == db.StatusSuspended {
		return failed, nil
	}
	access, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		utils.Logger.Error("oauth tokens", zap.String("provider", name), zap.Error(err))
		return failed, nil
	}
	fragment := url.Values{"access_token": {access}, "refresh_token": {refresh}}.Encode()
	return &api.OAuthRedirect{URL: frontend + "/auth/callback#" + fragment}, nil
}

// createOAuthUser registers a password-less account: the stored hash is of random bytes nobody knows.
func (s *authService) createOAuthUser(ctx context.Context, email, name string) (*db.User, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(secret)), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if runes := []rune(name); len(runes) > 100 {
		name = string(runes[:100])
	}
	return s.users.Create(ctx, &db.User{
		Name: name, Email: email, PasswordHash: string(hash),
		Role: db.RoleCandidate, Status: db.StatusActive,
	})
}

func (s *authService) BootstrapAdmin(ctx context.Context) error {
	if s.cfg.AdminEmail == "" || s.cfg.AdminPassword == "" {
		return nil
	}
	exists, err := s.users.AdminExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := validatePassword(s.cfg.AdminPassword); err != nil {
		return fmt.Errorf("ADMIN_PASSWORD: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.users.Create(ctx, &db.User{
		Name: "Admin", Email: normalizeEmail(s.cfg.AdminEmail), PasswordHash: string(hash),
		Role: db.RoleAdmin, Status: db.StatusActive,
	})
	return err
}
