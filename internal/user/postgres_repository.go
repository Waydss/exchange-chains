package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	uniqueViolationCode   = "23505"
	uniqueLoginConstraint = "users_login_key"
)

// PostgresRepository persists users through a pgx connection pool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL-backed user repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create inserts a user and returns database-generated public fields.
func (r *PostgresRepository) Create(ctx context.Context, params CreateUserParams) (User, error) {
	const query = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id, login, password_hash, name, public_rating, shadow_rating, created_at, updated_at`

	createdUser, err := scanUser(r.pool.QueryRow(ctx, query, params.Login, params.PasswordHash))
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == uniqueViolationCode &&
			postgresError.ConstraintName == uniqueLoginConstraint {
			return User{}, fmt.Errorf("create user: %w", ErrLoginAlreadyExists)
		}

		return User{}, fmt.Errorf("create user: %w", err)
	}

	return createdUser, nil
}

// GetByLogin returns a user with private authentication data.
func (r *PostgresRepository) GetByLogin(ctx context.Context, login string) (User, error) {
	const query = `
		SELECT id, login, password_hash, name, public_rating, shadow_rating, created_at, updated_at
		FROM users
		WHERE login = $1`

	foundUser, err := scanUser(r.pool.QueryRow(ctx, query, login))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by login: %w", err)
	}

	return foundUser, nil
}

// GetByID returns a user by its primary key.
func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (User, error) {
	const query = `
		SELECT id, login, password_hash, name, public_rating, shadow_rating, created_at, updated_at
		FROM users
		WHERE id = $1`

	foundUser, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by ID: %w", err)
	}

	return foundUser, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var foundUser User
	err := row.Scan(
		&foundUser.ID,
		&foundUser.Login,
		&foundUser.PasswordHash,
		&foundUser.Name,
		&foundUser.PublicRating,
		&foundUser.ShadowRating,
		&foundUser.CreatedAt,
		&foundUser.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	return foundUser, nil
}
