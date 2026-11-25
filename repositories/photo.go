package repositories

import (
	"context"
)

// PhotoRepository keeps the profile photo of a user, apart from the user row's other columns.
type PhotoRepository interface {
	// Set replaces the photo (ErrNotFound when the user does not exist).
	Set(ctx context.Context, userID string, jpeg []byte) error
	// Get returns the photo (ErrNotFound when the user has none).
	Get(ctx context.Context, userID string) ([]byte, error)
	// Delete removes the photo; removing none is not an error.
	Delete(ctx context.Context, userID string) error
}

type photoRepository struct {
	store *Store
}

func NewPhotoRepository(store *Store) PhotoRepository {
	return &photoRepository{store: store}
}

func (r *photoRepository) Set(ctx context.Context, userID string, jpeg []byte) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `UPDATE users SET photo = $2, photo_updated_at = now(), updated_at = now() WHERE id = $1`, userID, jpeg)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *photoRepository) Get(ctx context.Context, userID string) ([]byte, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	var data []byte
	err := r.store.Conn(ctx).QueryRow(ctx, `SELECT photo FROM users WHERE id = $1 AND photo IS NOT NULL`, userID).Scan(&data)
	return data, MapError(err)
}

func (r *photoRepository) Delete(ctx context.Context, userID string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	_, err := r.store.Conn(ctx).Exec(ctx, `UPDATE users SET photo = NULL, photo_updated_at = NULL, updated_at = now() WHERE id = $1`, userID)
	return MapError(err)
}
