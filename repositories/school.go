package repositories

import (
	"context"

	"9itchi-account-api/models/db"

	"github.com/jackc/pgx/v5"
)

const schoolColumns = `code, name_fr, name_en, name_ar, position, created_at, updated_at`

// SchoolRepository keeps the list of schools offered to students.
type SchoolRepository interface {
	// List returns every school in the order shown to students.
	List(ctx context.Context) ([]db.School, error)
	// Create appends a school; ErrConflict when the code is taken.
	Create(ctx context.Context, s *db.School) (*db.School, error)
	// Update replaces the names of a school; ErrNotFound when the code is unknown.
	Update(ctx context.Context, s *db.School) (*db.School, error)
	// Delete removes a school; ErrNotFound when the code is unknown. Users who chose it keep their value.
	Delete(ctx context.Context, code string) error
}

type schoolRepository struct {
	store *Store
}

func NewSchoolRepository(store *Store) SchoolRepository {
	return &schoolRepository{store: store}
}

func (r *schoolRepository) List(ctx context.Context) ([]db.School, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `SELECT `+schoolColumns+` FROM schools ORDER BY position, code`)
	if err != nil {
		return nil, MapError(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.School])
	return items, MapError(err)
}

func (r *schoolRepository) Create(ctx context.Context, s *db.School) (*db.School, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		INSERT INTO schools (code, name_fr, name_en, name_ar, position)
		VALUES ($1, $2, $3, $4, (SELECT COALESCE(MAX(position), 0) + 1 FROM schools))
		RETURNING `+schoolColumns, s.Code, s.NameFR, s.NameEN, s.NameAR)
	if err != nil {
		return nil, MapError(err)
	}
	out, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.School])
	return out, MapError(err)
}

func (r *schoolRepository) Update(ctx context.Context, s *db.School) (*db.School, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		UPDATE schools SET name_fr = $2, name_en = $3, name_ar = $4, updated_at = now()
		WHERE code = $1
		RETURNING `+schoolColumns, s.Code, s.NameFR, s.NameEN, s.NameAR)
	if err != nil {
		return nil, MapError(err)
	}
	out, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.School])
	return out, MapError(err)
}

func (r *schoolRepository) Delete(ctx context.Context, code string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `DELETE FROM schools WHERE code = $1`, code)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
