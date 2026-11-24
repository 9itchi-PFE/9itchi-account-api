package repositories

import (
	"context"

	"9itchi-account-api/models/db"

	"github.com/jackc/pgx/v5"
)

const userColumns = `id, name, email, password_hash, role, status, school, created_at, updated_at`

// UserFilter selects users for the admin list. Empty strings mean "no filter".
type UserFilter struct {
	Q      string
	Role   string
	Status string
	Limit  int
	Offset int
}

// UserRepository is the only layer that touches PostgreSQL for user data.
type UserRepository interface {
	Create(ctx context.Context, u *db.User) (*db.User, error)
	GetByID(ctx context.Context, id string) (*db.User, error)
	GetByEmail(ctx context.Context, email string) (*db.User, error)
	UpdateName(ctx context.Context, id, name string) (*db.User, error)
	// UpdateProfile sets the name and, when school is not nil, the school ("" clears it).
	UpdateProfile(ctx context.Context, id, name string, school *string) (*db.User, error)
	UpdateStatus(ctx context.Context, id, status string) (*db.User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f UserFilter) ([]db.User, int, error)
	AdminExists(ctx context.Context) (bool, error)
}

type userRepository struct {
	store *Store
}

func NewUserRepository(store *Store) UserRepository {
	return &userRepository{store: store}
}

func (r *userRepository) one(ctx context.Context, sql string, args ...any) (*db.User, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, MapError(err)
	}
	u, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.User])
	return u, MapError(err)
}

func (r *userRepository) Create(ctx context.Context, u *db.User) (*db.User, error) {
	return r.one(ctx, `INSERT INTO users (name, email, password_hash, role, status, school) VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+userColumns,
		u.Name, u.Email, u.PasswordHash, u.Role, u.Status, u.School)
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*db.User, error) {
	return r.one(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*db.User, error) {
	return r.one(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
}

func (r *userRepository) UpdateName(ctx context.Context, id, name string) (*db.User, error) {
	return r.one(ctx, `UPDATE users SET name = $2, updated_at = now() WHERE id = $1 RETURNING `+userColumns, id, name)
}

func (r *userRepository) UpdateProfile(ctx context.Context, id, name string, school *string) (*db.User, error) {
	set, value := school != nil, ""
	if school != nil {
		value = *school
	}
	return r.one(ctx, `UPDATE users SET name = $2, school = CASE WHEN $3 THEN NULLIF($4, '') ELSE school END, updated_at = now()
		WHERE id = $1 RETURNING `+userColumns, id, name, set, value)
}

func (r *userRepository) UpdateStatus(ctx context.Context, id, status string) (*db.User, error) {
	return r.one(ctx, `UPDATE users SET status = $2, updated_at = now() WHERE id = $1 RETURNING `+userColumns, id, status)
}

func (r *userRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// userWhere matches name/email by case-insensitive substring (strpos, so no LIKE wildcard escaping is needed).
const userWhere = `WHERE ($1::text = '' OR strpos(lower(name), lower($1::text)) > 0 OR strpos(email, lower($1::text)) > 0)
  AND ($2::text = '' OR role = $2::text) AND ($3::text = '' OR status = $3::text)`

func (r *userRepository) List(ctx context.Context, f UserFilter) ([]db.User, int, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	var total int
	if err := r.store.Conn(ctx).QueryRow(ctx, `SELECT count(*) FROM users `+userWhere, f.Q, f.Role, f.Status).Scan(&total); err != nil {
		return nil, 0, MapError(err)
	}
	rows, err := r.store.Conn(ctx).Query(ctx,
		`SELECT `+userColumns+` FROM users `+userWhere+` ORDER BY created_at DESC, id LIMIT $4 OFFSET $5`,
		f.Q, f.Role, f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, MapError(err)
	}
	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.User])
	if err != nil {
		return nil, 0, MapError(err)
	}
	return users, total, nil
}

func (r *userRepository) AdminExists(ctx context.Context) (bool, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	var exists bool
	err := r.store.Conn(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE role = 'admin')`).Scan(&exists)
	return exists, MapError(err)
}
