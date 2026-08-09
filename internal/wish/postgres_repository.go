package wish

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository reads wanted items through pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL wanted-item repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ListByUserID returns every wanted item owned by a user, newest first.
func (r *PostgresRepository) ListByUserID(ctx context.Context, userID int64) ([]Details, error) {
	const query = `
		SELECT w.id, w.user_id, w.category_id, w.price_min, w.price_max,
		       w.title_keywords, w.created_at, w.updated_at,
		       c.id, c.name, c.created_at
		FROM wishes AS w
		JOIN categories AS c ON c.id = w.category_id
		WHERE w.user_id = $1
		ORDER BY w.created_at DESC, w.id DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user wishes: %w", err)
	}
	defer rows.Close()

	wishes := make([]Details, 0)
	for rows.Next() {
		var found Details
		if err := rows.Scan(
			&found.Wish.ID,
			&found.Wish.UserID,
			&found.Wish.CategoryID,
			&found.Wish.PriceMin,
			&found.Wish.PriceMax,
			&found.Wish.TitleKeywords,
			&found.Wish.CreatedAt,
			&found.Wish.UpdatedAt,
			&found.Category.ID,
			&found.Category.Name,
			&found.Category.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user wish: %w", err)
		}
		wishes = append(wishes, found)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user wishes: %w", err)
	}

	return wishes, nil
}
