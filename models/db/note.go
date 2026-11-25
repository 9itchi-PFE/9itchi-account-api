package db

import "time"

// UserNote is what the team wrote about a student for itself.
type UserNote struct {
	UserID    string    `db:"user_id"`
	Note      string    `db:"note"`
	UpdatedBy string    `db:"updated_by"`
	UpdatedAt time.Time `db:"updated_at"`
}
