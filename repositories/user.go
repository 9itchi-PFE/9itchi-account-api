package repositories

import (
	"context"
	"fmt"

	"9itchi-account-api/models/db"

	"github.com/jackc/pgx/v5"
)

const userColumns = `id, name, email, password_hash, role, status, school, phone, city, speciality, year, linkedin, github, language, created_at, updated_at`

// ProfilePatch holds the optional fields of the profile; nil means unchanged, "" clears the field.
type ProfilePatch struct {
	School, Phone, City, Speciality, Year, LinkedIn, GitHub *string
}

type profileField struct {
	column string
	value  *string
}

func (p ProfilePatch) fields() []profileField {
	return []profileField{
		{"school", p.School}, {"phone", p.Phone}, {"city", p.City}, {"speciality", p.Speciality},
		{"year", p.Year}, {"linkedin", p.LinkedIn}, {"github", p.GitHub},
	}
}

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
	// UpdateProfile sets the name and each field of the patch that is not nil ("" clears it).
	UpdateProfile(ctx context.Context, id, name string, patch ProfilePatch) (*db.User, error)
	// UpdateLanguage sets the language 9itchi writes to the user in.
	UpdateLanguage(ctx context.Context, id, language string) (*db.User, error)
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
	return r.one(ctx, `INSERT INTO users (name, email, password_hash, role, status, school, language)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE(NULLIF($7, ''), 'fr')) RETURNING `+userColumns,
		u.Name, u.Email, u.PasswordHash, u.Role, u.Status, u.School, u.Language)
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

func (r *userRepository) UpdateProfile(ctx context.Context, id, name string, patch ProfilePatch) (*db.User, error) {
	args := []any{id, name}
	set := "name = $2"
	for _, f := range patch.fields() {
		if f.value == nil {
			continue
		}
		args = append(args, *f.value)
		// The column names come from fields(), never from the request.
		set += fmt.Sprintf(", %s = NULLIF($%d, '')", f.column, len(args))
	}
	return r.one(ctx, `UPDATE users SET `+set+`, updated_at = now() WHERE id = $1 RETURNING `+userColumns, args...)
}

func (r *userRepository) UpdateLanguage(ctx context.Context, id, language string) (*db.User, error) {
	return r.one(ctx, `UPDATE users SET language = $2, updated_at = now() WHERE id = $1 RETURNING `+userColumns, id, language)
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
