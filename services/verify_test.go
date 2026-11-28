package services

import (
	"context"
	"net/http"
	"testing"
	"time"

	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signedUp creates an account through Signup: unverified, with a code in the mailbox.
func signedUp(t *testing.T, e *authEnv) {
	t.Helper()
	_, apiErr := e.svc.Signup(context.Background(), api.SignupRequest{Name: "Ann", Email: "ann@example.com", Password: goodPassword})
	require.Nil(t, apiErr)
	require.NotEmpty(t, e.mailer.VerifyCode)
}

func wrongCode(code string) string {
	if code == "000000" {
		return "000001"
	}
	return "000000"
}

func TestVerifyEmailSignsTheUserIn(t *testing.T) {
	e := newAuthEnv(testCfg)
	signedUp(t, e)

	res, apiErr := e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: " ANN@example.com ", Code: e.mailer.VerifyCode})
	require.Nil(t, apiErr)
	assert.NotEmpty(t, res.AccessToken)
	assert.Contains(t, e.repo.Tokens, hashToken(res.RefreshToken))
	u, _ := e.users.GetByEmail(context.Background(), "ann@example.com")
	assert.True(t, u.EmailVerified)

	_, apiErr = e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: "ann@example.com", Code: e.mailer.VerifyCode})
	require.NotNil(t, apiErr, "a code works once")
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)

	login, apiErr := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	require.Nil(t, apiErr, "a verified account signs in")
	assert.NotEmpty(t, login.AccessToken)
}

func TestVerifyEmailRefusesABadCode(t *testing.T) {
	e := newAuthEnv(testCfg)
	signedUp(t, e)
	good := e.mailer.VerifyCode

	for i := 0; i < maxVerifyAttempts; i++ {
		_, apiErr := e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: "ann@example.com", Code: wrongCode(good)})
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
		assert.Equal(t, errInvalidResetMsg, apiErr.Error.Error())
	}
	_, apiErr := e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: "ann@example.com", Code: good})
	require.NotNil(t, apiErr, "after 5 wrong tries the code is dead")

	_, apiErr = e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: "nobody@example.com", Code: good})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code, "an unknown email looks like a wrong code")
}

func TestVerifyEmailRefusesAnExpiredCode(t *testing.T) {
	e := newAuthEnv(testCfg)
	signedUp(t, e)
	for _, v := range e.repo.Verifs {
		v.ExpiresAt = time.Now().Add(-time.Second)
	}
	_, apiErr := e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: "ann@example.com", Code: e.mailer.VerifyCode})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestUnverifiedLoginIsRefusedAndSendsACode(t *testing.T) {
	e := newAuthEnv(testCfg)
	signedUp(t, e)
	require.Equal(t, 1, e.mailer.Verifications)

	_, apiErr := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.Code)
	assert.Equal(t, errNotVerified, apiErr.Error.Error())
	assert.Equal(t, 1, e.mailer.Verifications, "no second email within a minute")

	for _, v := range e.repo.Verifs {
		v.CreatedAt = time.Now().Add(-2 * verifyResendCooldown)
	}
	_, apiErr = e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	require.NotNil(t, apiErr)
	assert.Equal(t, 2, e.mailer.Verifications, "a fresh code once the minute has passed")

	_, apiErr = e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: "Wrong0pass"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code, "a wrong password says nothing about verification")
	assert.Equal(t, 2, e.mailer.Verifications)
}

func TestUnverifiedRefreshIsRefused(t *testing.T) {
	e := newAuthEnv(testCfg)
	u := e.addUser(t, "ann@example.com", db.StatusActive)
	res, apiErr := e.svc.Login(context.Background(), api.LoginRequest{Email: "ann@example.com", Password: goodPassword})
	require.Nil(t, apiErr)
	u.EmailVerified = false // a session opened before verification was required

	_, apiErr = e.svc.RefreshToken(context.Background(), api.RefreshTokenRequest{RefreshToken: res.RefreshToken})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.Code)
	assert.Equal(t, errNotVerified, apiErr.Error.Error())
}

func TestResendVerification(t *testing.T) {
	e := newAuthEnv(testCfg)
	signedUp(t, e)
	first := e.mailer.VerifyCode

	res, apiErr := e.svc.ResendVerification(context.Background(), api.ResendVerificationRequest{Email: "ann@example.com"})
	require.Nil(t, apiErr)
	assert.Equal(t, resendMessage, res.Message)
	assert.Equal(t, 1, e.mailer.Verifications, "within a minute nothing is sent")

	for _, v := range e.repo.Verifs {
		v.CreatedAt = time.Now().Add(-2 * verifyResendCooldown)
	}
	_, apiErr = e.svc.ResendVerification(context.Background(), api.ResendVerificationRequest{Email: "ann@example.com"})
	require.Nil(t, apiErr)
	assert.Equal(t, 2, e.mailer.Verifications)
	if e.mailer.VerifyCode != first {
		_, apiErr = e.svc.VerifyEmail(context.Background(), api.VerifyEmailRequest{Email: "ann@example.com", Code: first})
		require.NotNil(t, apiErr, "the new code replaces the old one")
	}

	for _, email := range []string{"nobody@example.com"} {
		res, apiErr = e.svc.ResendVerification(context.Background(), api.ResendVerificationRequest{Email: email})
		require.Nil(t, apiErr)
		assert.Equal(t, resendMessage, res.Message, "same answer for an unknown email")
	}
	assert.Equal(t, 2, e.mailer.Verifications)
}

func TestResendVerificationSkipsVerifiedAndSuspended(t *testing.T) {
	e := newAuthEnv(testCfg)
	e.addUser(t, "ann@example.com", db.StatusActive)
	_, apiErr := e.svc.ResendVerification(context.Background(), api.ResendVerificationRequest{Email: "ann@example.com"})
	require.Nil(t, apiErr)
	assert.Zero(t, e.mailer.Verifications, "a verified account gets no code")

	e = newAuthEnv(testCfg)
	u := e.addUser(t, "ann@example.com", db.StatusSuspended)
	u.EmailVerified = false
	_, apiErr = e.svc.ResendVerification(context.Background(), api.ResendVerificationRequest{Email: "ann@example.com"})
	require.Nil(t, apiErr)
	assert.Zero(t, e.mailer.Verifications, "a suspended account gets no code")
}

func TestAdminsNeverNeedVerification(t *testing.T) {
	e := newAuthEnv(testCfg)
	u := e.addUser(t, "admin@example.com", db.StatusActive)
	u.Role, u.EmailVerified = db.RoleAdmin, false
	_, apiErr := e.svc.Login(context.Background(), api.LoginRequest{Email: "admin@example.com", Password: goodPassword})
	assert.Nil(t, apiErr)
}
