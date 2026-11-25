package services

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"9itchi-account-api/core"
	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/models/db"
	"9itchi-account-api/models/mapping"
	"9itchi-account-api/repositories"
	"9itchi-account-api/utils"

	"go.uber.org/zap"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// UserService holds all business logic for user. Handlers stay thin.
type UserService interface {
	// GET /users/me: current profile
	GetMe(ctx context.Context, userID string) (*api.UserResponse, *models.ApiError)
	// PUT /users/me: update profile (name only)
	UpdateMe(ctx context.Context, userID string, req api.UpdateMeRequest) (*api.UserResponse, *models.ApiError)
	// PUT /users/me/language: the language 9itchi writes to the caller in (emails, notifications)
	SetLanguage(ctx context.Context, userID string, req api.SetLanguageRequest) (*api.UserResponse, *models.ApiError)
	// DELETE /users/me: delete account and data (right to erasure)
	DeleteMe(ctx context.Context, userID string) (*api.MessageResponse, *models.ApiError)
	// GET /admin/users: list and search users
	ListUsers(ctx context.Context, userID string, req api.ListUsersRequest) (*api.ListUsersResponse, *models.ApiError)
	// GET /admin/users/:id: user detail
	GetUser(ctx context.Context, userID string, id string) (*api.UserResponse, *models.ApiError)
	// PUT /admin/users/:id/suspend: suspend or reactivate
	SuspendUser(ctx context.Context, userID string, id string, req api.SuspendUserRequest) (*api.UserResponse, *models.ApiError)
	// DELETE /admin/users/:id: delete user
	DeleteUser(ctx context.Context, userID string, id string) (*api.MessageResponse, *models.ApiError)
	// GET /internal/users/:id: user lookup for other services
	InternalGetUser(ctx context.Context, id string) (*api.InternalUserResponse, *models.ApiError)
}

type userService struct {
	repo   repositories.UserRepository
	tokens repositories.AuthRepository
	tx     repositories.Transactor
	// erasers are the other services that keep data about a user, in the order they are asked to erase it.
	erasers []core.UserEraser
}

func NewUserService(repo repositories.UserRepository, tokens repositories.AuthRepository, tx repositories.Transactor, erasers ...core.UserEraser) UserService {
	return &userService{repo: repo, tokens: tokens, tx: tx, erasers: erasers}
}

func notFound() *models.ApiError { return apiError(http.StatusNotFound, "user not found") }

// getByID fetches a user after validating the id format.
func (s *userService) getByID(ctx context.Context, id string) (*db.User, *models.ApiError) {
	if !isUUID(id) {
		return nil, badRequest("invalid user id")
	}
	u, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, internalError("user get", err)
	}
	return u, nil
}

func (s *userService) GetMe(ctx context.Context, userID string) (*api.UserResponse, *models.ApiError) {
	u, apiErr := s.getByID(ctx, userID)
	if apiErr != nil {
		return nil, apiErr
	}
	res := mapping.UserToResponse(u)
	return &res, nil
}

func (s *userService) SetLanguage(ctx context.Context, userID string, req api.SetLanguageRequest) (*api.UserResponse, *models.ApiError) {
	if !isUUID(userID) {
		return nil, badRequest("invalid user id")
	}
	if !languages[req.Language] {
		return nil, badRequest("language must be fr, en or ar")
	}
	u, err := s.repo.UpdateLanguage(ctx, userID, req.Language)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, internalError("user language", err)
	}
	res := mapping.UserToResponse(u)
	return &res, nil
}

// languages are the languages of the interface, and so the ones 9itchi can write to a user in.
var languages = map[string]bool{"fr": true, "en": true, "ar": true}

func (s *userService) UpdateMe(ctx context.Context, userID string, req api.UpdateMeRequest) (*api.UserResponse, *models.ApiError) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		return nil, badRequest("name must be between 1 and 100 characters")
	}
	if !isUUID(userID) {
		return nil, badRequest("invalid user id")
	}
	patch, apiErr := profilePatch(req)
	if apiErr != nil {
		return nil, apiErr
	}
	u, err := s.repo.UpdateProfile(ctx, userID, name, patch)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, internalError("user update", err)
	}
	res := mapping.UserToResponse(u)
	return &res, nil
}

// delete removes an account and everything 9itchi keeps about its owner. The other services erase their part
// first (the CV, the applications and the Gmail connection, the saved offers, the notifications, the support
// conversations, the email tracking; payments keeps the receipts and stops the subscription), and the account
// goes last: if one of them fails the account stays, nothing is half-promised, and the same request can be sent
// again, since erasing twice is harmless everywhere.
func (s *userService) delete(ctx context.Context, id string) (*api.MessageResponse, *models.ApiError) {
	if !isUUID(id) {
		return nil, badRequest("invalid user id")
	}
	if _, err := s.repo.GetByID(ctx, id); errors.Is(err, repositories.ErrNotFound) {
		return nil, notFound()
	} else if err != nil {
		return nil, internalError("user delete", err)
	}
	for _, e := range s.erasers {
		if err := e.EraseUser(ctx, id); err != nil {
			// The name of the service and the status only: never whose data it was.
			utils.Logger.Error("user delete: a service did not erase its part", zap.String("service", e.Name()), zap.Error(err))
			return nil, apiError(http.StatusBadGateway, "the account could not be deleted yet, try again in a moment")
		}
	}
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, internalError("user delete", err)
	}
	return &api.MessageResponse{Message: "user deleted"}, nil
}

func (s *userService) DeleteMe(ctx context.Context, userID string) (*api.MessageResponse, *models.ApiError) {
	return s.delete(ctx, userID)
}

func (s *userService) ListUsers(ctx context.Context, _ string, req api.ListUsersRequest) (*api.ListUsersResponse, *models.ApiError) {
	if req.Role != "" && req.Role != db.RoleCandidate && req.Role != db.RoleAdmin {
		return nil, badRequest("invalid role filter")
	}
	if req.Status != "" && req.Status != db.StatusActive && req.Status != db.StatusSuspended {
		return nil, badRequest("invalid status filter")
	}
	q := strings.TrimSpace(req.Q)
	if len(q) > 100 {
		return nil, badRequest("q too long")
	}
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	users, total, err := s.repo.List(ctx, repositories.UserFilter{
		Q: q, Role: req.Role, Status: req.Status, Limit: size, Offset: (page - 1) * size,
	})
	if err != nil {
		return nil, internalError("user list", err)
	}
	items := make([]api.UserResponse, 0, len(users))
	for i := range users {
		items = append(items, mapping.UserToResponse(&users[i]))
	}
	return &api.ListUsersResponse{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func (s *userService) GetUser(ctx context.Context, _ string, id string) (*api.UserResponse, *models.ApiError) {
	return s.GetMe(ctx, id)
}

func (s *userService) SuspendUser(ctx context.Context, userID string, id string, req api.SuspendUserRequest) (*api.UserResponse, *models.ApiError) {
	if req.Suspended == nil {
		return nil, badRequest("suspended is required")
	}
	if !isUUID(id) {
		return nil, badRequest("invalid user id")
	}
	id = strings.ToLower(id)
	if id == userID {
		return nil, apiError(http.StatusForbidden, "cannot suspend yourself")
	}
	status := db.StatusActive
	if *req.Suspended {
		status = db.StatusSuspended
	}
	var updated *db.User
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		u, err := s.repo.UpdateStatus(ctx, id, status)
		if err != nil {
			return err
		}
		updated = u
		if status == db.StatusSuspended {
			return s.tokens.RevokeAllRefreshTokens(ctx, id)
		}
		return nil
	})
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, internalError("user suspend", err)
	}
	res := mapping.UserToResponse(updated)
	return &res, nil
}

func (s *userService) DeleteUser(ctx context.Context, userID string, id string) (*api.MessageResponse, *models.ApiError) {
	if strings.EqualFold(id, userID) {
		return nil, apiError(http.StatusForbidden, "cannot delete yourself")
	}
	return s.delete(ctx, id)
}

func (s *userService) InternalGetUser(ctx context.Context, id string) (*api.InternalUserResponse, *models.ApiError) {
	u, apiErr := s.getByID(ctx, id)
	if apiErr != nil {
		return nil, apiErr
	}
	res := mapping.UserToInternal(u)
	return &res, nil
}
