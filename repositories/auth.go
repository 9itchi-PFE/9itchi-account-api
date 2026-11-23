package repositories

// AuthRepository is the only layer that touches PostgreSQL for auth data.
// Use store.Conn(ctx), a context from QueryTimeout, and return MapError(err).
type AuthRepository interface {
}

type authRepository struct {
	store *Store
}

func NewAuthRepository(store *Store) AuthRepository {
	return &authRepository{store: store}
}
