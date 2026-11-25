package handlers

import (
	"context"
	"net/http"
	"testing"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeUser struct {
	services.UserService
	list    func(api.ListUsersRequest) (*api.ListUsersResponse, *models.ApiError)
	suspend func(actor, id string, r api.SuspendUserRequest) (*api.UserResponse, *models.ApiError)
	update  func(api.UpdateMeRequest) (*api.UserResponse, *models.ApiError)
}

func (f fakeUser) SetLanguage(_ context.Context, _ string, r api.SetLanguageRequest) (*api.UserResponse, *models.ApiError) {
	return &api.UserResponse{Language: r.Language}, nil
}

func (f fakeUser) ListUsers(_ context.Context, _ string, r api.ListUsersRequest) (*api.ListUsersResponse, *models.ApiError) {
	return f.list(r)
}

func (f fakeUser) SuspendUser(_ context.Context, actor, id string, r api.SuspendUserRequest) (*api.UserResponse, *models.ApiError) {
	return f.suspend(actor, id, r)
}

func (f fakeUser) UpdateMe(_ context.Context, _ string, r api.UpdateMeRequest) (*api.UserResponse, *models.ApiError) {
	return f.update(r)
}

func TestListUsersHandlerBindsQuery(t *testing.T) {
	var got api.ListUsersRequest
	svc := fakeUser{list: func(r api.ListUsersRequest) (*api.ListUsersResponse, *models.ApiError) {
		got = r
		return &api.ListUsersResponse{Items: []api.UserResponse{}, Total: 0, Page: r.Page, PageSize: r.PageSize}, nil
	}}
	w := do(ListUsers(svc), http.MethodGet, "/admin/users?page=2&page_size=10&q=ann&role=admin&status=active", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, api.ListUsersRequest{Page: 2, PageSize: 10, Q: "ann", Role: "admin", Status: "active"}, got)
	assert.JSONEq(t, `{"items":[],"total":0,"page":2,"page_size":10}`, w.Body.String())

	w = do(ListUsers(svc), http.MethodGet, "/admin/users?page=abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSuspendUserHandler(t *testing.T) {
	var gotActor, gotID string
	var gotSuspended bool
	svc := fakeUser{suspend: func(actor, id string, r api.SuspendUserRequest) (*api.UserResponse, *models.ApiError) {
		gotActor, gotID, gotSuspended = actor, id, *r.Suspended
		return &api.UserResponse{ID: id, Status: "suspended"}, nil
	}}
	w := do(SuspendUser(svc), http.MethodPut, "/admin/users/x/suspend", `{"suspended":false}`, gin.Param{Key: "id", Value: "x"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "uid-1", gotActor)
	assert.Equal(t, "x", gotID)
	assert.False(t, gotSuspended, "false is accepted, not treated as missing")

	w = do(SuspendUser(svc), http.MethodPut, "/admin/users/x/suspend", `{}`, gin.Param{Key: "id", Value: "x"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMeHandlerValidates(t *testing.T) {
	svc := fakeUser{update: func(r api.UpdateMeRequest) (*api.UserResponse, *models.ApiError) {
		return &api.UserResponse{Name: r.Name}, nil
	}}
	w := do(UpdateMe(svc), http.MethodPut, "/users/me", `{"name":"Bob"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Bob"`)

	w = do(UpdateMe(svc), http.MethodPut, "/users/me", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// The form sends every field, empty when the student gave none: that clears them, it is not an error.
	w = do(UpdateMe(svc), http.MethodPut, "/users/me",
		`{"name":"Bob","school":"","phone":"","city":"","speciality":"","year":"","linkedin":"","github":""}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestSetLanguageHandlerValidates(t *testing.T) {
	w := do(SetLanguage(fakeUser{}), http.MethodPut, "/users/me/language", `{"language":"en"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"language":"en"`)

	for _, body := range []string{`{}`, `{"language":"de"}`, `{`} {
		w = do(SetLanguage(fakeUser{}), http.MethodPut, "/users/me/language", body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Contains(t, w.Body.String(), `"error"`)
	}
}
