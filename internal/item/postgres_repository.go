package item

import (
	"context"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/wish"
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

// ListByUserID returns every item owned by a user, newest first.
func (r *PostgresRepository) ListByUserID(ctx context.Context, userID int64) ([]Details, error) {
	return r.readItems(ctx, "WHERE i.user_id = $1", userID)
}

// GetByID returns one item with category and photo metadata.
func (r *PostgresRepository) GetByID(ctx context.Context, itemID int64) (Details, error) {
	items, err := r.readItems(ctx, "WHERE i.id = $1", itemID)
	if err != nil {
		return Details{}, err
	}
	if len(items) == 0 {
		return Details{}, ErrNotFound
	}

	return items[0], nil
}

// UpdatePhoto replaces the stored object key for one item photo.
func (r *PostgresRepository) UpdatePhoto(ctx context.Context, itemID, photoID int64, objectKey string) (Photo, error) {
	const query = `
		UPDATE item_photos
		SET object_key = $3
		WHERE item_id = $1 AND id = $2
		RETURNING id, item_id, object_key, position, created_at`

	var updated Photo
	err := r.pool.QueryRow(ctx, query, itemID, photoID, objectKey).Scan(
		&updated.ID,
		&updated.ItemID,
		&updated.ObjectKey,
		&updated.Position,
		&updated.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Photo{}, ErrPhotoNotFound
		}
		return Photo{}, fmt.Errorf("update item photo: %w", err)
	}

	return updated, nil
}

// DeletePhoto deletes one photo metadata row for an item.
func (r *PostgresRepository) DeletePhoto(ctx context.Context, itemID, photoID int64) error {
	const query = `
		DELETE FROM item_photos
		WHERE item_id = $1 AND id = $2`

	result, err := r.pool.Exec(ctx, query, itemID, photoID)
	if err != nil {
		return fmt.Errorf("delete item photo: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrPhotoNotFound
	}

	return nil
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		StatusSearching,
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
		INSERT INTO item_photos (item_id, object_key, position)
		VALUES ($1, $2, $3)
		RETURNING id, item_id, object_key, position, created_at`

	batch := &pgx.Batch{}
	for _, photo := range photos {
		batch.Queue(query, itemID, photo.ObjectKey, photo.Position)
	}

	results := t.tx.SendBatch(ctx, batch)
	for range photos {
		var created Photo
		if err := results.QueryRow().Scan(
			&created.ID,
			&created.ItemID,
			&created.ObjectKey,
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

// CreateWish inserts the wanted item associated with the current create request.
func (t *postgresTransaction) CreateWish(ctx context.Context, params CreateWishParams) (wish.WishedItem, error) {
	const query = `
		INSERT INTO wishes (user_id, category_id, price_min, price_max, title_keywords)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, category_id, price_min, price_max, title_keywords, created_at, updated_at`

	var created wish.WishedItem
	err := t.tx.QueryRow(
		ctx,
		query,
		params.UserID,
		params.CategoryID,
		params.PriceMin,
		params.PriceMax,
		params.TitleKeywords,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.CategoryID,
		&created.PriceMin,
		&created.PriceMax,
		&created.TitleKeywords,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return wish.WishedItem{}, fmt.Errorf("insert wish: %w", err)
	}

	return created, nil
}

func (r *PostgresRepository) readItems(ctx context.Context, where string, args ...any) ([]Details, error) {
	query := `
		SELECT i.id, i.user_id, i.title, i.description, i.condition,
		       i.category_id, i.price_min, i.price_max, i.wanted_category_id,
		       i.status, i.created_at, i.updated_at,
		       c.id, c.name, c.created_at,
		       wc.id, wc.name, wc.created_at
		FROM items AS i
		JOIN categories AS c ON c.id = i.category_id
		JOIN categories AS wc ON wc.id = i.wanted_category_id`
	if where != "" {
		query += " " + where
	}
	query += " ORDER BY i.created_at DESC, i.id DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}

	items := make([]Details, 0)
	itemIndex := make(map[int64]int)
	itemIDs := make([]int64, 0)
	for rows.Next() {
		var found Details
		if err := rows.Scan(
			&found.Item.ID,
			&found.Item.UserID,
			&found.Item.Title,
			&found.Item.Description,
			&found.Item.Condition,
			&found.Item.CategoryID,
			&found.Item.PriceMin,
			&found.Item.PriceMax,
			&found.Item.WantedCategoryID,
			&found.Item.Status,
			&found.Item.CreatedAt,
			&found.Item.UpdatedAt,
			&found.Category.ID,
			&found.Category.Name,
			&found.Category.CreatedAt,
			&found.WantedCategory.ID,
			&found.WantedCategory.Name,
			&found.WantedCategory.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan item: %w", err)
		}
		found.Photos = make([]Photo, 0)
		itemIndex[found.Item.ID] = len(items)
		itemIDs = append(itemIDs, found.Item.ID)
		items = append(items, found)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate items: %w", err)
	}
	rows.Close()
	if len(items) == 0 {
		return items, nil
	}

	const photoQuery = `
		SELECT id, item_id, object_key, position, created_at
		FROM item_photos
		WHERE item_id = ANY($1)
		ORDER BY item_id, position, id`

	photoRows, err := r.pool.Query(ctx, photoQuery, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("query item photos: %w", err)
	}
	defer photoRows.Close()

	for photoRows.Next() {
		var photo Photo
		if err := photoRows.Scan(
			&photo.ID,
			&photo.ItemID,
			&photo.ObjectKey,
			&photo.Position,
			&photo.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan item photo: %w", err)
		}
		index, ok := itemIndex[photo.ItemID]
		if ok {
			items[index].Photos = append(items[index].Photos, photo)
		}
	}
	if err := photoRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate item photos: %w", err)
	}

	return items, nil
}
