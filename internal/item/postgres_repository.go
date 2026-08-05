package item

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository persists items and photo metadata in PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL item repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// WithinTransaction runs all item writes in one PostgreSQL transaction.
func (r *PostgresRepository) WithinTransaction(ctx context.Context, fn func(Transaction) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin item transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(&postgresTransaction{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit item transaction: %w", err)
	}

	return nil
}

type postgresTransaction struct {
	tx pgx.Tx
}

func (t *postgresTransaction) CreateItem(ctx context.Context, params CreateItemParams) (Item, error) {
	const query = `
		INSERT INTO items (
			user_id,
			title,
			description,
			condition,
			category_id,
			price_min,
			price_max,
			wanted_category_id,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'searching')
		RETURNING id, user_id, title, description, condition, category_id,
			price_min, price_max, wanted_category_id, status, created_at, updated_at`

	var created Item
	err := t.tx.QueryRow(
		ctx,
		query,
		params.UserID,
		params.Title,
		params.Description,
		params.Condition,
		params.CategoryID,
		params.PriceMin,
		params.PriceMax,
		params.WantedCategoryID,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.Title,
		&created.Description,
		&created.Condition,
		&created.CategoryID,
		&created.PriceMin,
		&created.PriceMax,
		&created.WantedCategoryID,
		&created.Status,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Item{}, fmt.Errorf("insert item: %w", err)
	}

	return created, nil
}

func (t *postgresTransaction) CreatePhotos(
	ctx context.Context,
	itemID int64,
	photos []CreatePhotoParams,
) ([]Photo, error) {
	createdPhotos := make([]Photo, 0, len(photos))
	if len(photos) == 0 {
		return createdPhotos, nil
	}

	const query = `
		INSERT INTO item_photos (item_id, url, position)
		VALUES ($1, $2, $3)
		RETURNING id, item_id, url, position, created_at`

	batch := &pgx.Batch{}
	for _, photo := range photos {
		batch.Queue(query, itemID, photo.URL, photo.Position)
	}

	results := t.tx.SendBatch(ctx, batch)
	for range photos {
		var created Photo
		if err := results.QueryRow().Scan(
			&created.ID,
			&created.ItemID,
			&created.URL,
			&created.Position,
			&created.CreatedAt,
		); err != nil {
			_ = results.Close()
			return nil, fmt.Errorf("insert item photo: %w", err)
		}
		createdPhotos = append(createdPhotos, created)
	}
	if err := results.Close(); err != nil {
		return nil, fmt.Errorf("close item photo batch: %w", err)
	}

	return createdPhotos, nil
}
