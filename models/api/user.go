package api

import "time"

// UserResponse is the public view of a user (never the password hash).
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	School    string    `json:"school,omitempty"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InternalUserResponse is the minimal user view served to peer services.
type InternalUserResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
	// Language is the language to write to the user in (fr, en or ar).
	Language string `json:"language"`
}

// SetLanguageRequest is the validated body of SetLanguage.
type SetLanguageRequest struct {
	Language string `json:"language" binding:"required,oneof=fr en ar"`
}

// UpdateMeRequest is the validated body of UpdateMe (name only).
type UpdateMeRequest struct {
	Name string `json:"name" binding:"required,max=100"`
	// School, when present, replaces the school; "" clears it.
	School *string `json:"school" binding:"omitempty,max=120"`
}

// SuspendUserRequest is the validated body of SuspendUser.
type SuspendUserRequest struct {
	Suspended *bool `json:"suspended" binding:"required"`
}

// ListUsersRequest holds the query parameters of ListUsers.
type ListUsersRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Q        string `form:"q"`
	Role     string `form:"role"`
	Status   string `form:"status"`
}

// ListUsersResponse is a page of users.
type ListUsersResponse struct {
	Items    []UserResponse `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
