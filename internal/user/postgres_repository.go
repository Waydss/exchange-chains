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
		INSERT INTO users (login, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, login, password_hash, name, public_rating, shadow_rating, created_at, updated_at`

	createdUser, err := scanUser(r.pool.QueryRow(ctx, query, params.Login, params.PasswordHash, params.Name))
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

func (r *PostgresRepository) GetUserRatings(ctx context.Context, userId int64) ([]UserRating, error) {
	const query = `
		SELECT from_id, to_id, public_rating, barter_id
		FROM user_ratings
		WHERE to_id = $1
		ORDER BY id DESC`

	rows, err := r.pool.Query(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("list user ratings: %w", err)
	}
	defer rows.Close()

	ratings := make([]UserRating, 0)
	for rows.Next() {
		var rating UserRating
		if err := rows.Scan(
			&rating.FromId,
			&rating.ToId,
			&rating.PublicRating,
			&rating.BarterId,
		); err != nil {
			return nil, fmt.Errorf("scan user rating: %w", err)
		}
		ratings = append(ratings, rating)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user ratings: %w", err)
	}

	return ratings, nil
}

// ValidateBarterParticipants returns an error unless the barter exists and both users are its participants.
func (r *PostgresRepository) ValidateBarterParticipants(ctx context.Context, barterId int64, fromUserId int64, toUserId int64) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM chain_barters WHERE id = $1)`, barterId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check barter existence: %w", err)
	}
	if !exists {
		return ErrBarterNotFound
	}

	err = r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM chain_barters
			WHERE id = $1
			  AND ((from_user_id = $2 AND to_user_id = $3) OR (from_user_id = $3 AND to_user_id = $2))
		)`, barterId, fromUserId, toUserId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check barter participants: %w", err)
	}
	if !exists {
		return ErrBarterUsersMismatch
	}

	return nil
}

func (r *PostgresRepository) UpdatePublicRating(ctx context.Context, fromId int64, toId int64, rating float64, barterId int64) (average float64, total int, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin rating transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ratingExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_ratings
			WHERE from_id = $1 AND to_id = $2 AND barter_id = $3
		)`, fromId, toId, barterId).Scan(&ratingExists)
	if err != nil {
		return 0, 0, fmt.Errorf("check existing rating: %w", err)
	}
	if ratingExists {
		return 0, 0, ErrRatingAlreadyExists
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_ratings (from_id, to_id, public_rating, barter_id)
		VALUES ($1, $2, $3, $4)`, fromId, toId, rating, barterId)
	if err != nil {
		return 0, 0, fmt.Errorf("insert user rating: %w", err)
	}

	err = tx.QueryRow(ctx, `
		SELECT AVG(public_rating), COUNT(*)
		FROM user_ratings
		WHERE to_id = $1`, toId).Scan(&average, &total)
	if err != nil {
		return 0, 0, fmt.Errorf("compute average rating: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET public_rating = $1, updated_at = NOW()
		WHERE id = $2`, average, toId)
	if err != nil {
		return 0, 0, fmt.Errorf("update user public rating: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit rating transaction: %w", err)
	}

	return average, total, nil
}

// FIXME We currently don't have TTL on the user_shadow_registry table, so the shadow rating is permenant for each user for now

func (r *PostgresRepository) UpdateShadowRating(ctx context.Context, userId int64, reasonString string) error {
	amount, err := getShadowScore(reasonString)
	if err != nil {
		return fmt.Errorf("get shadow reason err: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin shadow rating transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		SELECT id, login, password_hash, name, public_rating, shadow_rating, created_at, updated_at
		FROM users
		WHERE id = $1`

	foundUser, err := scanUser(tx.QueryRow(ctx, query, userId))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("get user by ID: %w", err)
	}

	newShadowRating := min(foundUser.ShadowRating+amount, 10)

	_, err = tx.Exec(ctx, `
		INSERT INTO user_shadow_registry (user_id, reason, amount)
		VALUES ($1, $2, $3)`, userId, reasonString, amount)
	if err != nil {
		return fmt.Errorf("insert shadow rating record: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET shadow_rating = $1, updated_at = NOW()
		WHERE id = $2`, newShadowRating, userId)
	if err != nil {
		return fmt.Errorf("update user shadow rating: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit shadow rating transaction: %w", err)
	}

	return nil
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
