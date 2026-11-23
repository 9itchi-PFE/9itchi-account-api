package services

import (
	"context"

	"9itchi-account-api/models"
	"9itchi-account-api/models/api"
	"9itchi-account-api/repositories"
)

// UserService holds all business logic for user. Handlers stay thin.
type UserService interface {
	// GET /users/me: current profile
	GetMe(ctx context.Context, userID string) (any, *models.ApiError)
	// PUT /users/me: update profile
	UpdateMe(ctx context.Context, userID string, req api.UpdateMeRequest) (any, *models.ApiError)
	// DELETE /users/me: delete account and data (right to erasure)
	DeleteMe(ctx context.Context, userID string) (any, *models.ApiError)
	// GET /admin/users: list and search users
	ListUsers(ctx context.Context, userID string) (any, *models.ApiError)
	// GET /admin/users/:id: user detail
	GetUser(ctx context.Context, userID string, id string) (any, *models.ApiError)
	// PUT /admin/users/:id/suspend: suspend or reactivate
	SuspendUser(ctx context.Context, userID string, id string, req api.SuspendUserRequest) (any, *models.ApiError)
	// DELETE /admin/users/:id: delete user
	DeleteUser(ctx context.Context, userID string, id string) (any, *models.ApiError)
	// GET /internal/users/:id: user lookup for other services
	InternalGetUser(ctx context.Context, id string) (any, *models.ApiError)
}

type userService struct {
	repo repositories.UserRepository
}

func NewUserService(repo repositories.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetMe(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) UpdateMe(_ context.Context, _ string, _ api.UpdateMeRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) DeleteMe(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) ListUsers(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) GetUser(_ context.Context, _ string, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) SuspendUser(_ context.Context, _ string, _ string, _ api.SuspendUserRequest) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) DeleteUser(_ context.Context, _ string, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *userService) InternalGetUser(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}
