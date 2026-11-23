package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// fakeAuth overrides only the methods a test needs; others panic on the nil embedded interface.
type fakeAuth struct {
	services.AuthService
	signup func(api.SignupRequest) (*api.AuthResponse, *models.ApiError)
	logout func(string, api.LogoutRequest) (*api.MessageResponse, *models.ApiError)
}

func (f fakeAuth) Signup(_ context.Context, r api.SignupRequest) (*api.AuthResponse, *models.ApiError) {
	return f.signup(r)
}

func (f fakeAuth) Logout(_ context.Context, id string, r api.LogoutRequest) (*api.MessageResponse, *models.ApiError) {
	return f.logout(id, r)
}

func do(h gin.HandlerFunc, method, path, body string, params ...gin.Param) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	c.Set("user_id", "uid-1")
	h(c)
	return w
}

func TestSignupHandler(t *testing.T) {
	var got api.SignupRequest
	svc := fakeAuth{signup: func(r api.SignupRequest) (*api.AuthResponse, *models.ApiError) {
		got = r
		if r.Email == "dup@example.com" {
			return nil, &models.ApiError{Code: http.StatusConflict, Error: errors.New("email already registered")}
		}
		return &api.AuthResponse{AccessToken: "a", RefreshToken: "r"}, nil
	}}

	w := do(Signup(svc), http.MethodPost, "/auth/signup", `{"name":"Ann","email":"ann@example.com","password":"Passw0rdX"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{"user":{"id":"","name":"","email":"","role":"","status":"","plan_id":null,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},"access_token":"a","refresh_token":"r"}`, w.Body.String())
	assert.Equal(t, "ann@example.com", got.Email)

	w = do(Signup(svc), http.MethodPost, "/auth/signup", `{"name":"Ann","email":"dup@example.com","password":"Passw0rdX"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"email already registered"`)

	for _, body := range []string{`{}`, `not json`, `{"name":"Ann","email":"bad","password":"Passw0rdX"}`, `{"name":"Ann","email":"a@b.co","password":"short"}`} {
		w = do(Signup(svc), http.MethodPost, "/auth/signup", body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Contains(t, w.Body.String(), `"error"`)
	}
}

func TestLogoutHandlerBodyOptional(t *testing.T) {
	var gotUser string
	var gotReq api.LogoutRequest
	svc := fakeAuth{logout: func(id string, r api.LogoutRequest) (*api.MessageResponse, *models.ApiError) {
		gotUser, gotReq = id, r
		return &api.MessageResponse{Message: "logged out"}, nil
	}}

	w := do(Logout(svc), http.MethodPost, "/auth/logout", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "uid-1", gotUser)
	assert.Empty(t, gotReq.RefreshToken)

	w = do(Logout(svc), http.MethodPost, "/auth/logout", `{"refresh_token":"tok"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "tok", gotReq.RefreshToken)

	w = do(Logout(svc), http.MethodPost, "/auth/logout", `{bad`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
