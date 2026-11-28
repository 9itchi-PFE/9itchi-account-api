package db

import "time"

// School is one entry of the list of schools offered to students.
type School struct {
	Code      string    `db:"code"`
	NameFR    string    `db:"name_fr"`
	NameEN    string    `db:"name_en"`
	NameAR    string    `db:"name_ar"`
	Position  int       `db:"position"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
