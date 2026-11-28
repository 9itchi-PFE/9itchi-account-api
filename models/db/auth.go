package db

import "time"

// RefreshToken maps table refresh_tokens (only the sha256 hash of the token is stored).
type RefreshToken struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// PasswordReset maps table password_resets, EmailVerification table email_verifications: both keep a 6-digit
// code sent by email, of which only the hash is stored.
type (
	PasswordReset     = OneTimeCode
	EmailVerification = OneTimeCode
)

// OneTimeCode is one code sent by email.
type OneTimeCode struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	CodeHash  string     `db:"code_hash"`
	Attempts  int        `db:"attempts"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}
