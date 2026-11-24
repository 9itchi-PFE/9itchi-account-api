package db

import "time"

const (
	RoleCandidate = "candidate"
	RoleAdmin     = "admin"

	StatusActive    = "active"
	StatusSuspended = "suspended"

	// DefaultLanguage is used when the client names none.
	DefaultLanguage = "fr"
)

// User maps table users.
type User struct {
	ID           string  `db:"id" json:"id"`
	Name         string  `db:"name" json:"name"`
	Email        string  `db:"email" json:"email"`
	PasswordHash string  `db:"password_hash" json:"-"`
	Role         string  `db:"role" json:"role"`
	Status       string  `db:"status" json:"status"`
	School       *string `db:"school" json:"school"`
	// Language is the language 9itchi writes to the user in: fr, en or ar.
	Language  string    `db:"language" json:"language"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
