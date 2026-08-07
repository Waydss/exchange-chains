package simulation

import (
	"context"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/category"
	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/L-ka080/DDB-6-case/internal/user"
	"github.com/L-ka080/DDB-6-case/internal/wish"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository performs bulk seeding operations through pgx COPY.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL simulation repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// LoadCategories returns every catalogue category.
func (r *PostgresRepository) LoadCategories(ctx context.Context) ([]category.Category, error) {
	const query = `SELECT id, name, created_at FROM categories ORDER BY id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load categories: %w", err)
	}
	defer rows.Close()

	categories := make([]category.Category, 0)
	for rows.Next() {
		var item category.Category
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	return categories, nil
}

// LoadLastIDs returns the current maximum primary keys.
func (r *PostgresRepository) LoadLastIDs(ctx context.Context) (int64, int64, int64, error) {
	const query = `
		SELECT
			COALESCE((SELECT MAX(id) FROM users), 0),
			COALESCE((SELECT MAX(id) FROM items), 0),
			COALESCE((SELECT MAX(id) FROM wishes), 0)`

	var userID, itemID, wishID int64
	if err := r.pool.QueryRow(ctx, query).Scan(&userID, &itemID, &wishID); err != nil {
		return 0, 0, 0, fmt.Errorf("load last ids: %w", err)
	}

	return userID, itemID, wishID, nil
}

// BulkInsertUsers copies many users in one round trip.
func (r *PostgresRepository) BulkInsertUsers(ctx context.Context, users []user.User) error {
	if len(users) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(users))
	for _, u := range users {
		rows = append(rows, []any{
			u.ID, u.Login, u.PasswordHash, u.Name, u.PublicRating, u.ShadowRating, u.CreatedAt, u.UpdatedAt,
		})
	}

	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"users"},
		[]string{"id", "login", "password_hash", "name", "public_rating", "shadow_rating", "created_at", "updated_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy users: %w", err)
	}

	return nil
}

// BulkInsertItems copies many items in one round trip.
func (r *PostgresRepository) BulkInsertItems(ctx context.Context, items []item.Item) error {
	if len(items) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(items))
	for _, it := range items {
		rows = append(rows, []any{
			it.ID, it.UserID, it.Title, it.Description, it.Condition, it.CategoryID,
			it.PriceMin, it.PriceMax, it.WantedCategoryID, it.Status, it.CreatedAt, it.UpdatedAt,
		})
	}

	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"items"},
		[]string{
			"id", "user_id", "title", "description", "condition", "category_id",
			"price_min", "price_max", "wanted_category_id", "status", "created_at", "updated_at",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy items: %w", err)
	}

	return nil
}

// BulkInsertWishes copies many wishes in one round trip.
func (r *PostgresRepository) BulkInsertWishes(ctx context.Context, wishes []wish.WishedItem) error {
	if len(wishes) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(wishes))
	for _, w := range wishes {
		rows = append(rows, []any{
			w.ID, w.UserID, w.CategoryID, w.PriceMin, w.PriceMax, w.TitleKeywords, w.CreatedAt, w.UpdatedAt,
		})
	}

	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"wishes"},
		[]string{
			"id", "user_id", "category_id", "price_min", "price_max", "title_keywords", "created_at", "updated_at",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy wishes: %w", err)
	}

	return nil
}

// SyncSequences advances the serial sequences past the explicitly copied ids.
func (r *PostgresRepository) SyncSequences(ctx context.Context) error {
	for _, table := range []string{"users", "items", "wishes"} {
		query := fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1))`,
			table, table,
		)
		if _, err := r.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("sync %s sequence: %w", table, err)
		}
	}

	return nil
}

// ClearData removes all simulated rows and their dependents.
func (r *PostgresRepository) ClearData(ctx context.Context) error {
	const query = `TRUNCATE TABLE users, items, wishes RESTART IDENTITY CASCADE`
	if _, err := r.pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("clear simulated data: %w", err)
	}

	return nil
}
