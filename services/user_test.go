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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adminID = "22222222-2222-4222-8222-222222222222"
	userID  = "11111111-1111-4111-8111-111111111111"
)

type userEnv struct {
	svc    UserService
	users  *mocks.UserRepo
	tokens *mocks.AuthRepo
}

func newUserEnv() *userEnv {
	e := &userEnv{users: mocks.NewUserRepo(), tokens: mocks.NewAuthRepo()}
	e.svc = NewUserService(e.users, e.tokens, mocks.Transactor{})
	e.users.Add(&db.User{ID: userID, Name: "Ann", Email: "ann@example.com", PasswordHash: "x", Role: db.RoleCandidate, Status: db.StatusActive})
	e.users.Add(&db.User{ID: adminID, Name: "Root", Email: "root@example.com", PasswordHash: "x", Role: db.RoleAdmin, Status: db.StatusActive})
	return e
}

func farFuture() time.Time { return time.Now().Add(time.Hour) }

func TestGetMe(t *testing.T) {
	e := newUserEnv()
	res, apiErr := e.svc.GetMe(context.Background(), userID)
	require.Nil(t, apiErr)
	assert.Equal(t, "ann@example.com", res.Email)

	_, apiErr = e.svc.GetMe(context.Background(), "33333333-3333-4333-8333-333333333333")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	e.users.Err = errors.New("boom")
	_, apiErr = e.svc.GetMe(context.Background(), userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestUpdateMe(t *testing.T) {
	e := newUserEnv()
	res, apiErr := e.svc.UpdateMe(context.Background(), userID, api.UpdateMeRequest{Name: "  New Name "})
	require.Nil(t, apiErr)
	assert.Equal(t, "New Name", res.Name)
	assert.Equal(t, "ann@example.com", res.Email, "only name changes")

	enis, empty := " ENIS ", ""
	res, apiErr = e.svc.UpdateMe(context.Background(), userID, api.UpdateMeRequest{Name: "New Name", School: &enis})
	require.Nil(t, apiErr)
	assert.Equal(t, "ENIS", res.School)
	res, apiErr = e.svc.UpdateMe(context.Background(), userID, api.UpdateMeRequest{Name: "New Name"})
	require.Nil(t, apiErr)
	assert.Equal(t, "ENIS", res.School, "no school in the body keeps it")
	res, apiErr = e.svc.UpdateMe(context.Background(), userID, api.UpdateMeRequest{Name: "New Name", School: &empty})
	require.Nil(t, apiErr)
	assert.Empty(t, res.School, "an empty school clears it")

	_, apiErr = e.svc.UpdateMe(context.Background(), userID, api.UpdateMeRequest{Name: "   "})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)

	_, apiErr = e.svc.UpdateMe(context.Background(), "33333333-3333-4333-8333-333333333333", api.UpdateMeRequest{Name: "X"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
}

func TestDeleteMe(t *testing.T) {
	e := newUserEnv()
	_, apiErr := e.svc.DeleteMe(context.Background(), userID)
	require.Nil(t, apiErr)
	assert.NotContains(t, e.users.Users, userID)
	_, apiErr = e.svc.DeleteMe(context.Background(), userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
}

func TestListUsers(t *testing.T) {
	tests := []struct {
		name     string
		req      api.ListUsersRequest
		wantCode int
		want     api.ListUsersResponse
		offset   int
	}{
		{"defaults", api.ListUsersRequest{}, 0, api.ListUsersResponse{Page: 1, PageSize: 20}, 0},
		{"clamps page size", api.ListUsersRequest{Page: 3, PageSize: 1000}, 0, api.ListUsersResponse{Page: 3, PageSize: 100}, 200},
		{"filters", api.ListUsersRequest{Page: 2, PageSize: 5, Q: " ann ", Role: "admin", Status: "active"}, 0, api.ListUsersResponse{Page: 2, PageSize: 5}, 5},
		{"bad role", api.ListUsersRequest{Role: "root"}, http.StatusBadRequest, api.ListUsersResponse{}, 0},
		{"bad status", api.ListUsersRequest{Status: "banned"}, http.StatusBadRequest, api.ListUsersResponse{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newUserEnv()
			res, apiErr := e.svc.ListUsers(context.Background(), adminID, tt.req)
			if tt.wantCode != 0 {
				require.NotNil(t, apiErr)
				assert.Equal(t, tt.wantCode, apiErr.Code)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, tt.want.Page, res.Page)
			assert.Equal(t, tt.want.PageSize, res.PageSize)
			assert.Equal(t, 2, res.Total)
			assert.Len(t, res.Items, 2)
			assert.Equal(t, tt.offset, e.users.LastFilter.Offset)
			assert.Equal(t, tt.want.PageSize, e.users.LastFilter.Limit)
		})
	}
	e := newUserEnv()
	_, _ = e.svc.ListUsers(context.Background(), adminID, api.ListUsersRequest{Q: " ann ", Role: "admin"})
	assert.Equal(t, "ann", e.users.LastFilter.Q)
	assert.Equal(t, "admin", e.users.LastFilter.Role)
}

func TestGetUserAndInternal(t *testing.T) {
	e := newUserEnv()
	res, apiErr := e.svc.GetUser(context.Background(), adminID, userID)
	require.Nil(t, apiErr)
	assert.Equal(t, userID, res.ID)

	_, apiErr = e.svc.GetUser(context.Background(), adminID, "not-a-uuid")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)

	in, apiErr := e.svc.InternalGetUser(context.Background(), userID)
	require.Nil(t, apiErr)
	assert.Equal(t, api.InternalUserResponse{ID: userID, Name: "Ann", Email: "ann@example.com", Role: db.RoleCandidate, Status: db.StatusActive}, *in)

	_, apiErr = e.svc.InternalGetUser(context.Background(), "33333333-3333-4333-8333-333333333333")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
}

func TestSuspendUser(t *testing.T) {
	yes, no := true, false
	e := newUserEnv()
	require.NoError(t, e.tokens.CreateRefreshToken(context.Background(), userID, "h", farFuture()))

	res, apiErr := e.svc.SuspendUser(context.Background(), adminID, userID, api.SuspendUserRequest{Suspended: &yes})
	require.Nil(t, apiErr)
	assert.Equal(t, db.StatusSuspended, res.Status)
	assert.NotNil(t, e.tokens.Tokens["h"].RevokedAt, "sessions revoked on suspend")

	res, apiErr = e.svc.SuspendUser(context.Background(), adminID, userID, api.SuspendUserRequest{Suspended: &no})
	require.Nil(t, apiErr)
	assert.Equal(t, db.StatusActive, res.Status)

	_, apiErr = e.svc.SuspendUser(context.Background(), adminID, adminID, api.SuspendUserRequest{Suspended: &yes})
	require.NotNil(t, apiErr, "admin cannot suspend self")
	assert.Equal(t, http.StatusForbidden, apiErr.Code)

	_, apiErr = e.svc.SuspendUser(context.Background(), adminID, "33333333-3333-4333-8333-333333333333", api.SuspendUserRequest{Suspended: &yes})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	_, apiErr = e.svc.SuspendUser(context.Background(), adminID, userID, api.SuspendUserRequest{})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestDeleteUser(t *testing.T) {
	e := newUserEnv()
	_, apiErr := e.svc.DeleteUser(context.Background(), adminID, adminID)
	require.NotNil(t, apiErr, "admin cannot delete self")
	assert.Equal(t, http.StatusForbidden, apiErr.Code)

	_, apiErr = e.svc.DeleteUser(context.Background(), adminID, "bad")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)

	_, apiErr = e.svc.DeleteUser(context.Background(), adminID, userID)
	require.Nil(t, apiErr)
	_, apiErr = e.svc.DeleteUser(context.Background(), adminID, userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
}

func TestSetLanguage(t *testing.T) {
	e := newUserEnv()
	res, apiErr := e.svc.SetLanguage(context.Background(), userID, api.SetLanguageRequest{Language: "ar"})
	require.Nil(t, apiErr)
	assert.Equal(t, "ar", res.Language)

	me, apiErr := e.svc.GetMe(context.Background(), userID)
	require.Nil(t, apiErr)
	assert.Equal(t, "ar", me.Language, "the choice is kept")

	for _, bad := range []string{"", "de", "FR"} {
		_, apiErr = e.svc.SetLanguage(context.Background(), userID, api.SetLanguageRequest{Language: bad})
		require.NotNil(t, apiErr, bad)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	}
	_, apiErr = e.svc.SetLanguage(context.Background(), "nope", api.SetLanguageRequest{Language: "en"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	_, apiErr = e.svc.SetLanguage(context.Background(), "33333333-3333-4333-8333-333333333333", api.SetLanguageRequest{Language: "en"})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
}

// fakeEraser is one of the other services: it records who it was asked to erase.
type fakeEraser struct {
	name string
	log  *[]string
	err  error
}

func (f fakeEraser) Name() string { return f.name }
func (f fakeEraser) EraseUser(_ context.Context, id string) error {
	*f.log = append(*f.log, f.name+":"+id)
	return f.err
}

// The other services erase their part first and in order; the account goes last, and stays if one of them fails.
func TestDeleteErasesEverywhereFirst(t *testing.T) {
	var calls []string
	users, tokens := mocks.NewUserRepo(), mocks.NewAuthRepo()
	add := func() {
		users.Add(&db.User{ID: userID, Name: "Ann", Email: "ann@example.com", PasswordHash: "x", Role: db.RoleCandidate, Status: db.StatusActive})
	}
	add()
	email, resume := fakeEraser{name: "email", log: &calls}, fakeEraser{name: "resume", log: &calls}
	svc := NewUserService(users, tokens, mocks.Transactor{}, email, resume)

	_, apiErr := svc.DeleteMe(context.Background(), userID)
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"email:" + userID, "resume:" + userID}, calls)
	_, apiErr = svc.GetMe(context.Background(), userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)

	// An account that does not exist asks nothing of the other services.
	calls = nil
	_, apiErr = svc.DeleteMe(context.Background(), userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
	assert.Empty(t, calls)

	// One service fails: the ones after it are not asked, the account stays, and the answer names no service.
	add()
	failing := fakeEraser{name: "resume", log: &calls, err: assert.AnError}
	last := fakeEraser{name: "payments", log: &calls}
	svc = NewUserService(users, tokens, mocks.Transactor{}, email, failing, last)
	_, apiErr = svc.DeleteMe(context.Background(), userID)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadGateway, apiErr.Code)
	assert.NotContains(t, apiErr.Error.Error(), "resume")
	assert.Equal(t, []string{"email:" + userID, "resume:" + userID}, calls)
	_, apiErr = svc.GetMe(context.Background(), userID)
	require.Nil(t, apiErr, "the account is still there, so the student can try again")

	// Sent again once the service is back: everything is asked again, and the account goes.
	calls = nil
	svc = NewUserService(users, tokens, mocks.Transactor{}, email, resume, last)
	_, apiErr = svc.DeleteMe(context.Background(), userID)
	require.Nil(t, apiErr)
	assert.Len(t, calls, 3)

	// The team deleting a student goes the same way.
	add()
	calls = nil
	_, apiErr = svc.DeleteUser(context.Background(), adminID, userID)
	require.Nil(t, apiErr)
	assert.Len(t, calls, 3)
}
