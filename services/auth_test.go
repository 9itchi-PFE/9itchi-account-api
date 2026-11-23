package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
	"9itchi-account-api/services/mocks"
	"9itchi-account-api/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var testCfg = utils.AppConfig{
	SecretKey:     "0123456789abcdef0123456789abcdef",
	EncryptionKey: "fedcba9876543210fedcba9876543210",
}

const goodPassword = "Passw0rdX"

type authEnv struct {
	svc    AuthService
	users  *mocks.UserRepo
	repo   *mocks.AuthRepo
	mailer *mocks.Mailer
}

func newAuthEnv(cfg utils.AppConfig) *authEnv {
	e := &authEnv{users: mocks.NewUserRepo(), repo: mocks.NewAuthRepo(), mailer: &mocks.Mailer{}}
	e.svc = NewAuthService(cfg, e.users, e.repo, mocks.Transactor{}, e.mailer)
	return e
}

func (e *authEnv) addUser(t *testing.T, email, status string) *db.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(goodPassword), bcrypt.MinCost)
	require.NoError(t, err)
	return e.users.Add(&db.User{ID: "11111111-1111-4111-8111-111111111111", Name: "Ann", Email: email,
		PasswordHash: string(hash), Role: db.RoleCandidate, Status: status})
}

func TestSignup(t *testing.T) {
	tests := []struct {
		name     string
		req      api.SignupRequest
		setup    func(e *authEnv)
		wantCode int
	}{
		{"ok, email lower-cased", api.SignupRequest{Name: " Ann ", Email: "Ann@Example.COM", Password: goodPassword}, nil, 0},
		{"weak password", api.SignupRequest{Name: "Ann", Email: "a@b.co", Password: "password1"}, nil, http.StatusBadRequest},
		{"blank name", api.SignupRequest{Name: "  ", Email: "a@b.co", Password: goodPassword}, nil, http.StatusBadRequest},
		{"duplicate email, case-insensitive", api.SignupRequest{Name: "Ann", Email: "ANN@example.com", Password: goodPassword},
			func(e *authEnv) { e.addUser(t, "ann@example.com", db.StatusActive) }, http.StatusConflict},
		{"repo failure", api.SignupRequest{Name: "Ann", Email: "a@b.co", Password: goodPassword},
			func(e *authEnv) { e.users.Err = errors.New("boom") }, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newAuthEnv(testCfg)
			if tt.setup != nil {
				tt.setup(e)
			}
			res, apiErr := e.svc.Signup(context.Background(), tt.req)
			if tt.wantCode != 0 {
				require.NotNil(t, apiErr)
				assert.Equal(t, tt.wantCode, apiErr.Code)
				if tt.wantCode == http.StatusInternalServerError {
					assert.NotContains(t, apiErr.Error.Error(), "boom")
				}
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, "ann@example.com", res.User.Email)
			assert.Equal(t, "Ann", res.User.Name)
			assert.Equal(t, db.RoleCandidate, res.User.Role)
			assert.Equal(t, db.StatusActive, res.User.Status)

			access, err := utils.ParseToken(testCfg, res.AccessToken)
			require.NoError(t, err)
			assert.Equal(t, utils.TokenAccess, access.Type)
			assert.WithinDuration(t, time.Now().Add(accessTTL), access.ExpiresAt.Time, 5*time.Second)
			refresh, err := utils.ParseToken(testCfg, res.RefreshToken)
			require.NoError(t, err)
			assert.Equal(t, utils.TokenRefresh, refresh.Type)
			assert.WithinDuration(t, time.Now().Add(refreshTTL), refresh.ExpiresAt.Time, 5*time.Second)
			assert.Contains(t, e.repo.Tokens, hashToken(res.RefreshToken), "only the hash is stored")
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name     string
		req      api.LoginRequest
		status   string
		wantCode int
	}{
		{"ok", api.LoginRequest{Email: " ANN@example.com ", Password: goodPassword}, db.StatusActive, 0},
		{"wrong password", api.LoginRequest{Email: "ann@example.com", Password: "Wrong0pass"}, db.StatusActive, http.StatusUnauthorized},
		{"unknown email", api.LoginRequest{Email: "nobody@example.com", Password: goodPassword}, db.StatusActive, http.StatusUnauthorized},
		{"suspended", api.LoginRequest{Email: "ann@example.com", Password: goodPassword}, db.StatusSuspended, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newAuthEnv(testCfg)
			e.addUser(t, "ann@example.com", tt.status)
			res, apiErr := e.svc.Login(context.Background(), tt.req)
			if tt.wantCode != 0 {
				require.NotNil(t, apiErr)
				assert.Equal(t, tt.wantCode, apiErr.Code)
				if tt.wantCode == http.StatusUnauthorized {
					assert.Equal(t, "invalid credentials", apiErr.Error.Error())
				}
				return
			}
			require.Nil(t, apiErr)
			assert.NotEmpty(t, res.AccessToken)
			assert.NotEmpty(t, res.RefreshToken)
		})
	}
}

func TestLoginRepoFailure(t *testing.T) {
	e := newAuthEnv(testCfg)
	e.users.Err = errors.New("boom")
	_, apiErr := e.svc.Login(context.Background(), api.LoginRequest{Email: "a@b.co", Password: goodPassword})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestRefreshRotation(t *testing.T) {
	e := newAuthEnv(testCfg)
	e.addUser(t, "ann@example.com", db.StatusActive)
	login, apiErr := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	require.Nil(t, apiErr)

	res, apiErr := e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: login.RefreshToken})
	require.Nil(t, apiErr)
	assert.NotEqual(t, login.RefreshToken, res.RefreshToken)
	assert.NotNil(t, e.repo.Tokens[hashToken(login.RefreshToken)].RevokedAt, "old token revoked")

	_, apiErr = e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: login.RefreshToken})
	require.NotNil(t, apiErr, "old token cannot be reused")
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)

	_, apiErr = e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: res.RefreshToken})
	assert.Nil(t, apiErr, "new token works")
}

func TestRefreshRejects(t *testing.T) {
	e := newAuthEnv(testCfg)
	u := e.addUser(t, "ann@example.com", db.StatusActive)
	login, _ := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})

	_, apiErr := e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: "garbage"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)

	_, apiErr = e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: login.AccessToken})
	require.NotNil(t, apiErr, "access token is not a refresh token")
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)

	// valid signature and type but never stored
	unstored, err := utils.IssueToken(testCfg, utils.Claims{ID: u.ID, Type: utils.TokenRefresh}, time.Hour)
	require.NoError(t, err)
	_, apiErr = e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: unstored})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)

	// suspended user
	u.Status = db.StatusSuspended
	_, apiErr = e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: login.RefreshToken})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.Code)
}

func TestRefreshExpiredStoredToken(t *testing.T) {
	e := newAuthEnv(testCfg)
	e.addUser(t, "ann@example.com", db.StatusActive)
	login, _ := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	e.repo.Tokens[hashToken(login.RefreshToken)].ExpiresAt = time.Now().Add(-time.Minute)
	_, apiErr := e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: login.RefreshToken})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)
}

func TestLogout(t *testing.T) {
	e := newAuthEnv(testCfg)
	u := e.addUser(t, "ann@example.com", db.StatusActive)
	a, _ := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	b, _ := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})

	_, apiErr := e.svc.Logout(context.Background(), u.ID, api.LogoutRequest{RefreshToken: a.RefreshToken})
	require.Nil(t, apiErr)
	assert.NotNil(t, e.repo.Tokens[hashToken(a.RefreshToken)].RevokedAt)
	assert.Nil(t, e.repo.Tokens[hashToken(b.RefreshToken)].RevokedAt)

	_, apiErr = e.svc.Logout(context.Background(), u.ID, api.LogoutRequest{})
	require.Nil(t, apiErr)
	assert.NotNil(t, e.repo.Tokens[hashToken(b.RefreshToken)].RevokedAt, "no token given revokes all")

	_, apiErr = e.svc.Logout(context.Background(), "not-a-uuid", api.LogoutRequest{})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)

	e.repo.Err = errors.New("boom")
	_, apiErr = e.svc.Logout(context.Background(), u.ID, api.LogoutRequest{})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestForgotPassword(t *testing.T) {
	e := newAuthEnv(testCfg)
	u := e.addUser(t, "ann@example.com", db.StatusActive)

	res, apiErr := e.svc.ForgotPassword(context.Background(), api.ForgotPasswordRequest{Email: "Nobody@example.com"})
	require.Nil(t, apiErr)
	unknownMsg := res.Message
	assert.Empty(t, e.mailer.Code, "no mail for unknown email")

	res, apiErr = e.svc.ForgotPassword(context.Background(), api.ForgotPasswordRequest{Email: "ANN@example.com"})
	require.Nil(t, apiErr)
	assert.Equal(t, unknownMsg, res.Message, "same body whether the account exists or not")
	assert.Regexp(t, `^\d{6}$`, e.mailer.Code)
	require.Len(t, e.repo.Resets, 1)
	for _, r := range e.repo.Resets {
		assert.NotEqual(t, e.mailer.Code, r.CodeHash, "code is hashed")
		assert.Equal(t, hashCode(testCfg.SecretKey, u.ID, e.mailer.Code), r.CodeHash)
		assert.WithinDuration(t, time.Now().Add(resetCodeTTL), r.ExpiresAt, 5*time.Second)
	}

	e.repo.Err = errors.New("boom")
	res, apiErr = e.svc.ForgotPassword(context.Background(), api.ForgotPasswordRequest{Email: "ann@example.com"})
	require.Nil(t, apiErr, "failures never leak")
	assert.Equal(t, unknownMsg, res.Message)
}

func TestForgotPasswordSuspendedGetsNoCode(t *testing.T) {
	e := newAuthEnv(testCfg)
	e.addUser(t, "ann@example.com", db.StatusSuspended)
	_, apiErr := e.svc.ForgotPassword(context.Background(), api.ForgotPasswordRequest{Email: "ann@example.com"})
	require.Nil(t, apiErr)
	assert.Empty(t, e.mailer.Code)
}

func TestResetPassword(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T) (*authEnv, *db.User, string) {
		e := newAuthEnv(testCfg)
		u := e.addUser(t, "ann@example.com", db.StatusActive)
		_, apiErr := e.svc.ForgotPassword(ctx, api.ForgotPasswordRequest{Email: u.Email})
		require.Nil(t, apiErr)
		return e, u, e.mailer.Code
	}

	t.Run("success revokes tokens, single use", func(t *testing.T) {
		e, u, code := setup(t)
		login, _ := e.svc.Login(ctx, api.LoginRequest{Email: u.Email, Password: goodPassword})
		_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: "ANN@example.com", Code: code, NewPassword: "NewPassw0rd"})
		require.Nil(t, apiErr)
		assert.NotNil(t, e.repo.Tokens[hashToken(login.RefreshToken)].RevokedAt)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("NewPassw0rd")))

		_, apiErr = e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: code, NewPassword: "Another0ne1"})
		require.NotNil(t, apiErr, "code cannot be reused")
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	})

	t.Run("wrong code", func(t *testing.T) {
		e, u, code := setup(t)
		wrong := "000000"
		if code == wrong {
			wrong = "111111"
		}
		_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: wrong, NewPassword: "NewPassw0rd"})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	})

	t.Run("expired", func(t *testing.T) {
		e, u, code := setup(t)
		for _, r := range e.repo.Resets {
			r.ExpiresAt = time.Now().Add(-time.Second)
		}
		_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: code, NewPassword: "NewPassw0rd"})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	})

	t.Run("locked after too many attempts", func(t *testing.T) {
		e, u, code := setup(t)
		wrong := "000000"
		if code == wrong {
			wrong = "111111"
		}
		for i := 0; i < maxResetAttempts; i++ {
			_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: wrong, NewPassword: "NewPassw0rd"})
			require.NotNil(t, apiErr)
		}
		_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: code, NewPassword: "NewPassw0rd"})
		require.NotNil(t, apiErr, "correct code rejected once locked")
	})

	t.Run("unknown email and weak password", func(t *testing.T) {
		e, u, code := setup(t)
		_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: "nobody@example.com", Code: code, NewPassword: "NewPassw0rd"})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
		_, apiErr = e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: code, NewPassword: "weakpass"})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	})

	t.Run("repo failure", func(t *testing.T) {
		e, u, code := setup(t)
		e.repo.Err = errors.New("boom")
		_, apiErr := e.svc.ResetPassword(ctx, api.ResetPasswordRequest{Email: u.Email, Code: code, NewPassword: "NewPassw0rd"})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	})
}

func TestGoogleStaysNotImplemented(t *testing.T) {
	e := newAuthEnv(testCfg)
	_, apiErr := e.svc.GoogleSignIn(context.Background())
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotImplemented, apiErr.Code)
	_, apiErr = e.svc.GoogleCallback(context.Background())
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotImplemented, apiErr.Code)
}

func TestBootstrapAdmin(t *testing.T) {
	ctx := context.Background()
	cfg := testCfg
	cfg.AdminEmail, cfg.AdminPassword = "Root@Example.com", goodPassword

	t.Run("creates admin once", func(t *testing.T) {
		e := newAuthEnv(cfg)
		require.NoError(t, e.svc.BootstrapAdmin(ctx))
		u, err := e.users.GetByEmail(ctx, "root@example.com")
		require.NoError(t, err)
		assert.Equal(t, db.RoleAdmin, u.Role)
		require.NoError(t, e.svc.BootstrapAdmin(ctx))
		assert.Len(t, e.users.Users, 1)
	})
	t.Run("skips when env unset", func(t *testing.T) {
		e := newAuthEnv(testCfg)
		require.NoError(t, e.svc.BootstrapAdmin(ctx))
		assert.Empty(t, e.users.Users)
	})
	t.Run("skips when an admin exists", func(t *testing.T) {
		e := newAuthEnv(cfg)
		e.users.Add(&db.User{Email: "other@example.com", Role: db.RoleAdmin})
		require.NoError(t, e.svc.BootstrapAdmin(ctx))
		assert.Len(t, e.users.Users, 1)
	})
	t.Run("rejects weak password", func(t *testing.T) {
		weak := cfg
		weak.AdminPassword = "weak"
		assert.Error(t, newAuthEnv(weak).svc.BootstrapAdmin(ctx))
	})
}

func TestValidatePassword(t *testing.T) {
	for pw, ok := range map[string]bool{
		"Passw0rd": true, "short1A": false, "alllowercase1": false, "ALLUPPERCASE1": false, "NoDigitsHere": false,
	} {
		assert.Equal(t, ok, validatePassword(pw) == nil, pw)
	}
}
