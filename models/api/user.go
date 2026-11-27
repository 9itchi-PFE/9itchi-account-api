package api

import "time"

// UserResponse is the public view of a user (never the password hash).
type UserResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	School     string    `json:"school,omitempty"`
	Phone      string    `json:"phone,omitempty"`
	City       string    `json:"city,omitempty"`
	Speciality string    `json:"speciality,omitempty"`
	Year       string    `json:"year,omitempty"`
	LinkedIn   string    `json:"linkedin,omitempty"`
	GitHub     string    `json:"github,omitempty"`
	HasPhoto   bool      `json:"has_photo"`
	Language   string    `json:"language"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
	// EmailVerified says whether a Google or LinkedIn sign-in vouched for the email (payments-api free grants).
	EmailVerified bool `json:"email_verified"`
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
	// The rest of the profile works the same way: a key that is present replaces the value, "" clears it, no key
	// keeps it. Year is a short code; LinkedIn and GitHub are the address of the student's page.
	Phone      *string `json:"phone" binding:"omitempty,max=30"`
	City       *string `json:"city" binding:"omitempty,max=80"`
	Speciality *string `json:"speciality" binding:"omitempty,max=120"`
	Year       *string `json:"year" binding:"omitempty,max=10"`
	LinkedIn   *string `json:"linkedin" binding:"omitempty,max=200"`
	GitHub     *string `json:"github" binding:"omitempty,max=200"`
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

// SetUserNoteRequest is the validated body of SetNote. An empty note empties it.
type SetUserNoteRequest struct {
	Note string `json:"note" binding:"max=8000"`
}

// UserNoteResponse is the internal note of the team about a user. UpdatedAt is nil when nothing was ever written.
type UserNoteResponse struct {
	UserID    string     `json:"user_id"`
	Note      string     `json:"note"`
	UpdatedAt *time.Time `json:"updated_at"`
}
