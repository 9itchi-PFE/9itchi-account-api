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
}

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
	return r.store.WithTx(ctx, func(ctx context.Context) error {
		ctx, cancel := QueryTimeout(ctx)
		defer cancel()
		conn := r.store.Conn(ctx)
		if _, err := conn.Exec(ctx, `DELETE FROM password_resets WHERE user_id = $1`, userID); err != nil {
			return MapError(err)
		}
		_, err := conn.Exec(ctx,
			`INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt)
		return MapError(err)
	})
}

func (r *authRepository) GetLatestPasswordReset(ctx context.Context, userID string) (*db.PasswordReset, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx,
		`SELECT id, user_id, code_hash, attempts, expires_at, used_at, created_at FROM password_resets
		 WHERE user_id = $1 AND used_at IS NULL ORDER BY created_at DESC LIMIT 1`, userID)
	if err != nil {
		return nil, MapError(err)
	}
	p, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.PasswordReset])
	return p, MapError(err)
}

func (r *authRepository) RecordResetAttempt(ctx context.Context, id string) (int, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	var attempts int
	err := r.store.Conn(ctx).QueryRow(ctx,
		`UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts`, id).Scan(&attempts)
	return attempts, MapError(err)
}

func (r *authRepository) MarkPasswordResetUsed(ctx context.Context, id string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `UPDATE password_resets SET used_at = now() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
