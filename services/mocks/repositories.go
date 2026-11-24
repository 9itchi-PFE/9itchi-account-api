// Package mocks holds hand-written in-memory fakes of the repository interfaces for service tests.
package mocks

import (
	"context"
	"fmt"
	"time"

	"9itchi-account-api/core"
	"9itchi-account-api/models/db"
	"9itchi-account-api/repositories"
)

// Transactor runs fn directly (no real transaction).
type Transactor struct{}

func (Transactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// UserRepo is an in-memory UserRepository. Err, when set, is returned by every method.
type UserRepo struct {
	Err        error
	Users      map[string]*db.User
	LastFilter repositories.UserFilter
	seq        int
}

func NewUserRepo() *UserRepo { return &UserRepo{Users: map[string]*db.User{}} }

// Add inserts a user with a generated uuid-shaped id when none is set.
func (m *UserRepo) Add(u *db.User) *db.User {
	if u.ID == "" {
		m.seq++
		u.ID = fmt.Sprintf("00000000-0000-4000-8000-%012d", m.seq)
	}
	m.Users[u.ID] = u
	return u
}

func (m *UserRepo) find(pred func(*db.User) bool) (*db.User, error) {
	for _, u := range m.Users {
		if pred(u) {
			c := *u
			return &c, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (m *UserRepo) Create(_ context.Context, u *db.User) (*db.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if _, err := m.find(func(x *db.User) bool { return x.Email == u.Email }); err == nil {
		return nil, repositories.ErrConflict
	}
	c := *u
	m.Add(&c)
	r := c
	return &r, nil
}

func (m *UserRepo) GetByID(_ context.Context, id string) (*db.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.find(func(x *db.User) bool { return x.ID == id })
}

func (m *UserRepo) GetByEmail(_ context.Context, email string) (*db.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.find(func(x *db.User) bool { return x.Email == email })
}

func (m *UserRepo) UpdateName(_ context.Context, id, name string) (*db.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	u, ok := m.Users[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	u.Name = name
	c := *u
	return &c, nil
}

func (m *UserRepo) UpdateStatus(_ context.Context, id, status string) (*db.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	u, ok := m.Users[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	u.Status = status
	c := *u
	return &c, nil
}

func (m *UserRepo) UpdatePassword(_ context.Context, id, hash string) error {
	if m.Err != nil {
		return m.Err
	}
	u, ok := m.Users[id]
	if !ok {
		return repositories.ErrNotFound
	}
	u.PasswordHash = hash
	return nil
}

func (m *UserRepo) Delete(_ context.Context, id string) error {
	if m.Err != nil {
		return m.Err
	}
	if _, ok := m.Users[id]; !ok {
		return repositories.ErrNotFound
	}
	delete(m.Users, id)
	return nil
}

func (m *UserRepo) List(_ context.Context, f repositories.UserFilter) ([]db.User, int, error) {
	m.LastFilter = f
	if m.Err != nil {
		return nil, 0, m.Err
	}
	var out []db.User
	for _, u := range m.Users {
		out = append(out, *u)
	}
	return out, len(out), nil
}

func (m *UserRepo) AdminExists(_ context.Context) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	_, err := m.find(func(x *db.User) bool { return x.Role == db.RoleAdmin })
	return err == nil, nil
}

// AuthRepo is an in-memory AuthRepository. Err, when set, is returned by every method.
type AuthRepo struct {
	Err     error
	Tokens  map[string]*db.RefreshToken  // by hash
	Resets  map[string]*db.PasswordReset // by id
	nextRst int
}

func NewAuthRepo() *AuthRepo {
	return &AuthRepo{Tokens: map[string]*db.RefreshToken{}, Resets: map[string]*db.PasswordReset{}}
}

func (m *AuthRepo) CreateRefreshToken(_ context.Context, userID, hash string, exp time.Time) error {
	if m.Err != nil {
		return m.Err
	}
	m.Tokens[hash] = &db.RefreshToken{UserID: userID, TokenHash: hash, ExpiresAt: exp}
	return nil
}

func (m *AuthRepo) ConsumeRefreshToken(_ context.Context, hash string) (*db.RefreshToken, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	t, ok := m.Tokens[hash]
	if !ok || t.RevokedAt != nil || !t.ExpiresAt.After(time.Now()) {
		return nil, repositories.ErrNotFound
	}
	now := time.Now()
	t.RevokedAt = &now
	c := *t
	return &c, nil
}

func (m *AuthRepo) RevokeRefreshToken(_ context.Context, userID, hash string) error {
	if m.Err != nil {
		return m.Err
	}
	if t, ok := m.Tokens[hash]; ok && t.UserID == userID && t.RevokedAt == nil {
		now := time.Now()
		t.RevokedAt = &now
	}
	return nil
}

func (m *AuthRepo) RevokeAllRefreshTokens(_ context.Context, userID string) error {
	if m.Err != nil {
		return m.Err
	}
	now := time.Now()
	for _, t := range m.Tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *AuthRepo) CreatePasswordReset(_ context.Context, userID, codeHash string, exp time.Time) error {
	if m.Err != nil {
		return m.Err
	}
	for id, r := range m.Resets {
		if r.UserID == userID {
			delete(m.Resets, id)
		}
	}
	m.nextRst++
	id := string(rune('a' + m.nextRst))
	m.Resets[id] = &db.PasswordReset{ID: id, UserID: userID, CodeHash: codeHash, ExpiresAt: exp}
	return nil
}

func (m *AuthRepo) GetLatestPasswordReset(_ context.Context, userID string) (*db.PasswordReset, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, r := range m.Resets {
		if r.UserID == userID && r.UsedAt == nil {
			c := *r
			return &c, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (m *AuthRepo) RecordResetAttempt(_ context.Context, id string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	r, ok := m.Resets[id]
	if !ok {
		return 0, repositories.ErrNotFound
	}
	r.Attempts++
	return r.Attempts, nil
}

func (m *AuthRepo) MarkPasswordResetUsed(_ context.Context, id string) error {
	if m.Err != nil {
		return m.Err
	}
	r, ok := m.Resets[id]
	if !ok || r.UsedAt != nil {
		return repositories.ErrNotFound
	}
	now := time.Now()
	r.UsedAt = &now
	return nil
}

// Mailer records the last reset code instead of sending it.
type Mailer struct {
	Err   error
	Email string
	Code  string
}

func (m *Mailer) SendPasswordResetCode(_ context.Context, email, code string) error {
	m.Email, m.Code = email, code
	return m.Err
}

// GoogleOAuth is a func-field fake of core.GoogleOAuth; it never touches the network.
type GoogleOAuth struct {
	AuthURLFn  func(state string) string
	ExchangeFn func(ctx context.Context, code string) (core.GoogleProfile, error)
}

func (g GoogleOAuth) AuthURL(state string) string { return g.AuthURLFn(state) }

func (g GoogleOAuth) Exchange(ctx context.Context, code string) (core.GoogleProfile, error) {
	return g.ExchangeFn(ctx, code)
}
