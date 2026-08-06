package cb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/chain_builder/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository reads and writes chain-builder data through pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a pgx-backed chain-builder repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// GetAccount returns an account together with all of its wishes and gives.
func (r *PostgresRepository) GetAccount(ctx context.Context, accountID uint) (entities.Account, error) {
	const accountQuery = `
		SELECT id, name, public_rating, shadow_rating
		FROM accounts
		WHERE id = $1`

	var account entities.Account
	var accountIDValue int64
	if err := r.pool.QueryRow(ctx, accountQuery, int64(accountID)).Scan(
		&accountIDValue,
		&account.Name,
		&account.PublicRating,
		&account.ShadowRating,
	); err != nil {
		return entities.Account{}, fmt.Errorf("get account: %w", err)
	}
	account.Id = uint(accountIDValue)

	wishes, err := r.getWishes(ctx, []uint{accountID}, nil)
	if err != nil {
		return entities.Account{}, err
	}
	gives, err := r.getGives(ctx, []uint{accountID})
	if err != nil {
		return entities.Account{}, err
	}

	account.Wishes = wishes[accountID]
	account.Gives = gives[accountID]
	return account, nil
}

// FindCandidateAccounts returns accounts whose wishes match the supplied give.
func (r *PostgresRepository) FindCandidateAccounts(ctx context.Context, give entities.GivedItem, limit int) ([]entities.Account, error) {
	const query = `
		SELECT DISTINCT a.id, a.name, a.public_rating, a.shadow_rating
		FROM accounts AS a
		JOIN wished_items AS w ON w.user_id = a.id
		WHERE w.category = $1
		  AND w.min_price <= $2
		  AND w.max_price >= $2
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, give.Category, int64(give.Price), limit)
	if err != nil {
		return nil, fmt.Errorf("find candidate accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]entities.Account, 0)
	accountIDs := make([]uint, 0)
	for rows.Next() {
		var account entities.Account
		var accountID int64
		if err := rows.Scan(&accountID, &account.Name, &account.PublicRating, &account.ShadowRating); err != nil {
			return nil, fmt.Errorf("scan candidate account: %w", err)
		}
		account.Id = uint(accountID)
		accounts = append(accounts, account)
		accountIDs = append(accountIDs, account.Id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidate accounts: %w", err)
	}
	if len(accounts) == 0 {
		return accounts, nil
	}

	wishFilter := &wishQueryFilter{Category: give.Category, Price: give.Price}
	wishes, err := r.getWishes(ctx, accountIDs, wishFilter)
	if err != nil {
		return nil, err
	}
	gives, err := r.getGives(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		accounts[i].Wishes = wishes[accounts[i].Id]
		accounts[i].Gives = gives[accounts[i].Id]
	}

	return accounts, nil
}

// CreateChain inserts a chain and ignores an already persisted signature.
func (r *PostgresRepository) CreateChain(ctx context.Context, chain entities.Chain) error {
	rootWish, err := json.Marshal(chain.RootWish)
	if err != nil {
		return fmt.Errorf("marshal root wish: %w", err)
	}
	barters, err := json.Marshal(chain.Barters)
	if err != nil {
		return fmt.Errorf("marshal barters: %w", err)
	}

	const query = `
		INSERT INTO chains (user_id, root_id, root_wish, barters, signature, complete, score)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (signature) DO NOTHING`
	if _, err := r.pool.Exec(
		ctx,
		query,
		int64(chain.UserId),
		int64(chain.RootId),
		rootWish,
		barters,
		chain.Signature,
		chain.Complete,
		chain.Score,
	); err != nil {
		return fmt.Errorf("create chain: %w", err)
	}

	return nil
}

type wishQueryFilter struct {
	Category entities.Category
	Price    uint
}

func (r *PostgresRepository) getWishes(ctx context.Context, accountIDs []uint, filter *wishQueryFilter) (map[uint][]entities.WishedItem, error) {
	query := `
		SELECT id, user_id, name, category, min_price, max_price, created_at, updated_at
		FROM wished_items
		WHERE user_id = ANY($1)`
	arguments := []any{toInt64IDs(accountIDs)}
	if filter != nil {
		query += ` AND category = $2 AND min_price <= $3 AND max_price >= $3`
		arguments = append(arguments, filter.Category, int64(filter.Price))
	}

	rows, err := r.pool.Query(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("get account wishes: %w", err)
	}
	defer rows.Close()

	result := make(map[uint][]entities.WishedItem, len(accountIDs))
	for rows.Next() {
		var wish entities.WishedItem
		var wishID, userID, minPrice, maxPrice int64
		if err := rows.Scan(
			&wishID,
			&userID,
			&wish.Name,
			&wish.Category,
			&minPrice,
			&maxPrice,
			&wish.CreatedAt,
			&wish.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account wish: %w", err)
		}
		wish.ID = uint(wishID)
		wish.UserId = uint(userID)
		wish.MinPrice = uint(minPrice)
		wish.MaxPrice = uint(maxPrice)
		result[wish.UserId] = append(result[wish.UserId], wish)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account wishes: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) getGives(ctx context.Context, accountIDs []uint) (map[uint][]entities.GivedItem, error) {
	const query = `
		SELECT id, user_id, name, category, price, created_at, updated_at
		FROM gived_items
		WHERE user_id = ANY($1)
		ORDER BY updated_at`

	rows, err := r.pool.Query(ctx, query, toInt64IDs(accountIDs))
	if err != nil {
		return nil, fmt.Errorf("get account gives: %w", err)
	}
	defer rows.Close()

	result := make(map[uint][]entities.GivedItem, len(accountIDs))
	for rows.Next() {
		var give entities.GivedItem
		var giveID, userID, price int64
		if err := rows.Scan(
			&giveID,
			&userID,
			&give.Name,
			&give.Category,
			&price,
			&give.CreatedAt,
			&give.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account give: %w", err)
		}
		give.ID = uint(giveID)
		give.UserId = uint(userID)
		give.Price = uint(price)
		result[give.UserId] = append(result[give.UserId], give)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account gives: %w", err)
	}

	return result, nil
}

func toInt64IDs(ids []uint) []int64 {
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = int64(id)
	}
	return result
}
