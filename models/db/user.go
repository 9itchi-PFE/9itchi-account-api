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
	// The rest of the profile, all optional: Year is a short code (l3, m1, m2, ing1, ing2, ing3, other).
	Phone      *string `db:"phone" json:"phone"`
	City       *string `db:"city" json:"city"`
	Speciality *string `db:"speciality" json:"speciality"`
	Year       *string `db:"year" json:"year"`
	LinkedIn   *string `db:"linkedin" json:"linkedin"`
	GitHub     *string `db:"github" json:"github"`
	// HasPhoto says whether the user has a profile photo (the photo is read on its own).
	HasPhoto bool `db:"has_photo" json:"has_photo"`
	// Language is the language 9itchi writes to the user in: fr, en or ar.
	Language  string    `db:"language" json:"language"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
