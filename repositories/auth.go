package repositories

import (
	"context"
	"time"

	"9itchi-account-api/models/db"

	"github.com/jackc/pgx/v5"
)

// AuthRepository is the only layer that touches PostgreSQL for refresh tokens and password resets.
type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	// ConsumeRefreshToken atomically revokes a stored, unrevoked, unexpired token. ErrNotFound otherwise.
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (*db.RefreshToken, error)
	// RevokeRefreshToken revokes one token of the user (idempotent).
	RevokeRefreshToken(ctx context.Context, userID, tokenHash string) error
	RevokeAllRefreshTokens(ctx context.Context, userID string) error
	// CreatePasswordReset replaces any previous reset of the user.
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	// GetLatestPasswordReset returns the newest unused reset. Expiry is enforced by the service.
	GetLatestPasswordReset(ctx context.Context, userID string) (*db.PasswordReset, error)
	// RecordResetAttempt increments the failed-attempt counter and returns it.
	RecordResetAttempt(ctx context.Context, id string) (int, error)
	// MarkPasswordResetUsed marks the reset used. ErrNotFound if it was already used.
	MarkPasswordResetUsed(ctx context.Context, id string) error
	// The email verification codes work like the reset codes: one live code per user, a few attempts, single use.
	CreateEmailVerification(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetLatestEmailVerification(ctx context.Context, userID string) (*db.EmailVerification, error)
	RecordVerificationAttempt(ctx context.Context, id string) (int, error)
	MarkEmailVerificationUsed(ctx context.Context, id string) error
}

// The two tables of codes sent by email; the names are constants, never input.
const (
	resetsTable        = "password_resets"
	verificationsTable = "email_verifications"
)

type authRepository struct {
	store *Store
}

func NewAuthRepository(store *Store) AuthRepository {
	return &authRepository{store: store}
}

func (r *authRepository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	_, err := r.store.Conn(ctx).Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
	return MapError(err)
}

func (r *authRepository) ConsumeRefreshToken(ctx context.Context, tokenHash string) (*db.RefreshToken, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx,
		`UPDATE refresh_tokens SET revoked_at = now()
		 WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		 RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`, tokenHash)
	if err != nil {
		return nil, MapError(err)
	}
	t, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.RefreshToken])
	return t, MapError(err)
}

func (r *authRepository) RevokeRefreshToken(ctx context.Context, userID, tokenHash string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	_, err := r.store.Conn(ctx).Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND token_hash = $2 AND revoked_at IS NULL`, userID, tokenHash)
	return MapError(err)
}

func (r *authRepository) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	_, err := r.store.Conn(ctx).Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return MapError(err)
}

func (r *authRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	return r.createCode(ctx, resetsTable, userID, codeHash, expiresAt)
}

func (r *authRepository) GetLatestPasswordReset(ctx context.Context, userID string) (*db.PasswordReset, error) {
	return r.latestCode(ctx, resetsTable, userID)
}

func (r *authRepository) RecordResetAttempt(ctx context.Context, id string) (int, error) {
	return r.recordAttempt(ctx, resetsTable, id)
}

func (r *authRepository) MarkPasswordResetUsed(ctx context.Context, id string) error {
	return r.markUsed(ctx, resetsTable, id)
}

func (r *authRepository) CreateEmailVerification(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	return r.createCode(ctx, verificationsTable, userID, codeHash, expiresAt)
}

func (r *authRepository) GetLatestEmailVerification(ctx context.Context, userID string) (*db.EmailVerification, error) {
	return r.latestCode(ctx, verificationsTable, userID)
}

func (r *authRepository) RecordVerificationAttempt(ctx context.Context, id string) (int, error) {
	return r.recordAttempt(ctx, verificationsTable, id)
}

func (r *authRepository) MarkEmailVerificationUsed(ctx context.Context, id string) error {
	return r.markUsed(ctx, verificationsTable, id)
}

// createCode replaces any previous code of the user in table.
func (r *authRepository) createCode(ctx context.Context, table, userID, codeHash string, expiresAt time.Time) error {
	return r.store.WithTx(ctx, func(ctx context.Context) error {
		ctx, cancel := QueryTimeout(ctx)
		defer cancel()
		conn := r.store.Conn(ctx)
		if _, err := conn.Exec(ctx, `DELETE FROM `+table+` WHERE user_id = $1`, userID); err != nil {
			return MapError(err)
		}
		_, err := conn.Exec(ctx,
			`INSERT INTO `+table+` (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt)
		return MapError(err)
	})
}

// latestCode returns the newest unused code of the user. Expiry is enforced by the service.
func (r *authRepository) latestCode(ctx context.Context, table, userID string) (*db.OneTimeCode, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx,
		`SELECT id, user_id, code_hash, attempts, expires_at, used_at, created_at FROM `+table+`
		 WHERE user_id = $1 AND used_at IS NULL ORDER BY created_at DESC LIMIT 1`, userID)
	if err != nil {
		return nil, MapError(err)
	}
	p, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.OneTimeCode])
	return p, MapError(err)
}

// recordAttempt increments the failed-attempt counter of a code and returns it.
func (r *authRepository) recordAttempt(ctx context.Context, table, id string) (int, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	var attempts int
	err := r.store.Conn(ctx).QueryRow(ctx,
		`UPDATE `+table+` SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts`, id).Scan(&attempts)
	return attempts, MapError(err)
}

// markUsed marks a code used. ErrNotFound if it was already used.
func (r *authRepository) markUsed(ctx context.Context, table, id string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `UPDATE `+table+` SET used_at = now() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
