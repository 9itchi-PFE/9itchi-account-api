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
	verifyCodeTTL          = 15 * time.Minute
	maxVerifyAttempts      = 5
	verifyResendCooldown   = time.Minute
	errNotVerified         = "email not verified"
	errInvalidCreds        = "invalid credentials"
	errInvalidRefresh      = "invalid refresh token"
	errInvalidResetMsg     = "invalid or expired code"
	errGoogleUnavailable   = "google sign-in unavailable"
	errLinkedInUnavailable = "linkedin sign-in unavailable"
)

// AuthService holds all business logic for auth. Handlers stay thin.
type AuthService interface {
	// POST /auth/signup: create a candidate account and email it a verification code (no session until verified)
	Signup(ctx context.Context, req api.SignupRequest) (*api.VerificationRequiredResponse, *models.ApiError)
	// POST /auth/verify-email: check the code, mark the email verified and sign the user in
	VerifyEmail(ctx context.Context, req api.VerifyEmailRequest) (*api.AuthResponse, *models.ApiError)
	// POST /auth/resend-verification: send a new code (always a generic answer)
	ResendVerification(ctx context.Context, req api.ResendVerificationRequest) (*api.MessageResponse, *models.ApiError)
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
	events   core.Publisher
}

// NewAuthService builds the service. events may be nil: registrations are then only logged.
func NewAuthService(cfg utils.AppConfig, users repositories.UserRepository, repo repositories.AuthRepository, tx repositories.Transactor, mailer core.Mailer, google, linkedin core.OAuthProvider, events core.Publisher) AuthService {
	if events == nil {
		events = core.LogPublisher{}
	}
	return &authService{cfg: cfg, users: users, repo: repo, tx: tx, mailer: mailer, google: google, linkedin: linkedin, events: events}
}

// registered tells the other services a student account now exists. The payload is the id only: no name, no email.
// A failure is logged and never undoes the signup.
func (s *authService) registered(ctx context.Context, user *db.User) {
	if err := s.events.Publish(ctx, core.EventUserRegistered, map[string]any{"user_id": user.ID}); err != nil {
		utils.Logger.Warn("publish event", zap.String("event", core.EventUserRegistered), zap.Error(err))
	}
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
	claims := utils.Claims{ID: u.ID, Email: u.Email, Role: u.Role, Type: utils.TokenAccess, EmailVerified: u.EmailVerified}
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

// needsVerification is true for an account that has not proved it owns its email: a password signup before its code.
// Admins are created by the platform and never need it.
func needsVerification(u *db.User) bool { return !u.EmailVerified && u.Role != db.RoleAdmin }

// sendVerification emails a new code to an unverified user, unless one was sent less than a minute ago (a user
// clicking twice, or a login loop, must not flood the mailbox). Failures are logged only.
func (s *authService) sendVerification(ctx context.Context, user *db.User) {
	if last, err := s.repo.GetLatestEmailVerification(ctx, user.ID); err == nil && time.Since(last.CreatedAt) < verifyResendCooldown {
		return
	}
	code, err := newCode()
	if err != nil {
		utils.Logger.Error("verify code", zap.Error(err))
		return
	}
	if err := s.repo.CreateEmailVerification(ctx, user.ID, hashCode(s.cfg.SecretKey, user.ID, code), time.Now().Add(verifyCodeTTL)); err != nil {
		utils.Logger.Error("verify store", zap.Error(err))
		return
	}
	if err := s.mailer.SendVerificationCode(ctx, user.Email, user.Language, code); err != nil {
		utils.Logger.Error("verify mail", zap.Error(err))
	}
}

// newCode draws a 6-digit code.
func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *authService) Signup(ctx context.Context, req api.SignupRequest) (*api.VerificationRequiredResponse, *models.ApiError) {
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
		Role: db.RoleCandidate, Status: db.StatusActive, School: school, Language: req.Language,
	})
	if errors.Is(err, repositories.ErrConflict) {
		return nil, apiError(http.StatusConflict, "email already registered")
	}
	if err != nil {
		return nil, internalError("signup create", err)
	}
	// The account exists for the other services only once its email is verified (VerifyEmail publishes it).
	s.sendVerification(ctx, user)
	return &api.VerificationRequiredResponse{Email: user.Email, VerificationRequired: true}, nil
}

const resendMessage = "if the account needs it, a new code has been sent"

func (s *authService) ResendVerification(ctx context.Context, req api.ResendVerificationRequest) (*api.MessageResponse, *models.ApiError) {
	generic := &api.MessageResponse{Message: resendMessage}
	user, err := s.users.GetByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if !errors.Is(err, repositories.ErrNotFound) {
			utils.Logger.Error("resend lookup", zap.Error(err))
		}
		return generic, nil
	}
	if user.Status == db.StatusActive && needsVerification(user) {
		s.sendVerification(ctx, user)
	}
	return generic, nil
}

func (s *authService) VerifyEmail(ctx context.Context, req api.VerifyEmailRequest) (*api.AuthResponse, *models.ApiError) {
	invalid := badRequest(errInvalidResetMsg)
	user, err := s.users.GetByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, internalError("verify lookup", err)
	}
	if user.Status == db.StatusSuspended {
		return nil, apiError(http.StatusForbidden, "account suspended")
	}
	if !needsVerification(user) {
		return nil, invalid
	}
	v, err := s.repo.GetLatestEmailVerification(ctx, user.ID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, internalError("verify load", err)
	}
	if v.Attempts >= maxVerifyAttempts || !time.Now().Before(v.ExpiresAt) {
		return nil, invalid
	}
	if subtle.ConstantTimeCompare([]byte(v.CodeHash), []byte(hashCode(s.cfg.SecretKey, user.ID, req.Code))) != 1 {
		if _, err := s.repo.RecordVerificationAttempt(ctx, v.ID); err != nil {
			return nil, internalError("verify attempt", err)
		}
		return nil, invalid
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.MarkEmailVerificationUsed(ctx, v.ID); err != nil {
			return err
		}
		verified, err := s.users.MarkEmailVerified(ctx, user.ID)
		if err == nil {
			user = verified
		}
		return err
	})
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, internalError("verify apply", err)
	}
	s.registered(ctx, user)
	access, refresh, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, internalError("verify tokens", err)
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
	if needsVerification(user) {
		// The password is right: send a fresh code so the screen that follows can be used at once.
		s.sendVerification(ctx, user)
		return nil, apiError(http.StatusForbidden, errNotVerified)
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
	if needsVerification(user) {
		// A session opened before verification was required ends here.
		return nil, apiError(http.StatusForbidden, errNotVerified)
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
	code, err := newCode()
	if err != nil {
		utils.Logger.Error("forgot code", zap.Error(err))
		return generic, nil
	}
	if err := s.repo.CreatePasswordReset(ctx, user.ID, hashCode(s.cfg.SecretKey, user.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		utils.Logger.Error("forgot store", zap.Error(err))
		return generic, nil
	}
	if err := s.mailer.SendPasswordResetCode(ctx, user.Email, user.Language, code); err != nil {
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
	if !user.EmailVerified {
		// The provider vouched for the email: from now on the account counts as owning it.
		if user, err = s.users.MarkEmailVerified(ctx, user.ID); err != nil {
			utils.Logger.Error("oauth verify email", zap.String("provider", name), zap.Error(err))
			return failed, nil
		}
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
	user, err := s.users.Create(ctx, &db.User{
		Name: name, Email: email, PasswordHash: string(hash),
		Role: db.RoleCandidate, Status: db.StatusActive,
	})
	if err != nil {
		return nil, err
	}
	s.registered(ctx, user)
	return user, nil
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
	admin, err := s.users.Create(ctx, &db.User{
		Name: "Admin", Email: normalizeEmail(s.cfg.AdminEmail), PasswordHash: string(hash),
		Role: db.RoleAdmin, Status: db.StatusActive,
	})
	if err != nil {
		return err
	}
	_, err = s.users.MarkEmailVerified(ctx, admin.ID)
	return err
}
