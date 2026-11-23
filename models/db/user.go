package db

import "time"

const (
	RoleCandidate = "candidate"
	RoleAdmin     = "admin"

	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// User maps table users.
type User struct {
	ID           string    `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	Status       string    `db:"status" json:"status"`
	PlanID       *string   `db:"plan_id" json:"plan_id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
