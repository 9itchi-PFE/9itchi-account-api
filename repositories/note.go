package repositories

import (
	"context"

	"9itchi-account-api/models/db"

	"github.com/jackc/pgx/v5"
)

// NoteRepository keeps the internal note of the team about a user.
type NoteRepository interface {
	// Get returns ErrNotFound when nothing was written about the user.
	Get(ctx context.Context, userID string) (*db.UserNote, error)
	// Set writes the note (replacing the one before); ErrNotFound when the user does not exist.
	Set(ctx context.Context, userID, note, by string) (*db.UserNote, error)
}

type noteRepository struct {
	store *Store
}

func NewNoteRepository(store *Store) NoteRepository {
	return &noteRepository{store: store}
}

func (r *noteRepository) Get(ctx context.Context, userID string) (*db.UserNote, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `SELECT user_id, note, updated_by, updated_at FROM user_notes WHERE user_id = $1`, userID)
	if err != nil {
		return nil, MapError(err)
	}
	n, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.UserNote])
	return n, MapError(err)
}

func (r *noteRepository) Set(ctx context.Context, userID, note, by string) (*db.UserNote, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	// The note is only written for a user who exists: the SELECT returns no row otherwise, and nothing is inserted.
	rows, err := r.store.Conn(ctx).Query(ctx, `
		INSERT INTO user_notes (user_id, note, updated_by)
		SELECT id, $2, $3 FROM users WHERE id = $1
		ON CONFLICT (user_id) DO UPDATE SET note = EXCLUDED.note, updated_by = EXCLUDED.updated_by, updated_at = now()
		RETURNING user_id, note, updated_by, updated_at`, userID, note, by)
	if err != nil {
		return nil, MapError(err)
	}
	n, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.UserNote])
	return n, MapError(err)
}
