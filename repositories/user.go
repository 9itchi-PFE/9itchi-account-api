package repositories

// UserRepository is the only layer that touches PostgreSQL for user data.
// Use store.Conn(ctx), a context from QueryTimeout, and return MapError(err).
type UserRepository interface {
}

type userRepository struct {
	store *Store
}

func NewUserRepository(store *Store) UserRepository {
	return &userRepository{store: store}
}
