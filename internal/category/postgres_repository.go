package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository reads categories from PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL category repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// GetByName performs a case-insensitive category lookup.
func (r *PostgresRepository) GetByName(ctx context.Context, name string) (Category, error) {
	const query = `
		SELECT id, name, created_at
		FROM categories
		WHERE LOWER(name) = LOWER($1)`

	var found Category
	err := r.pool.QueryRow(ctx, query, name).Scan(&found.ID, &found.Name, &found.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category by name: %w", err)
	}

	return found, nil
}
