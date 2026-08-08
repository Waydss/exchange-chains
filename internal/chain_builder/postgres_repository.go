package cb

import (
	"context"
	"errors"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MARK: Internal

// PostgresRepository reads and writes chain-builder data through pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a pgx-backed chain-builder repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// MARK: Public functions

// GetAccount returns an account together with all of its wishes and gives.
func (r *PostgresRepository) GetAccount(ctx context.Context, accountID int64) (userCandidate, error) {
	const accountQuery = `
		SELECT id, public_rating, shadow_rating
		FROM users
		WHERE id = $1`

	var account userCandidate
	if err := r.pool.QueryRow(ctx, accountQuery, accountID).Scan(
		&account.ID,
		&account.PublicRating,
		&account.ShadowRating,
	); err != nil {
		return userCandidate{}, fmt.Errorf("get account: %w", err)
	}

	wishes, err := r.getWishes(ctx, []int64{accountID}, nil)
	if err != nil {
		return userCandidate{}, err
	}
	gives, err := r.getGives(ctx, []int64{accountID})
	if err != nil {
		return userCandidate{}, err
	}

	account.Wishes = wishes[accountID]
	account.Gives = gives[accountID]
	return account, nil
}

// GetAccountGives returns all searching gives of the given account.
func (r *PostgresRepository) GetAccountGives(ctx context.Context, accountID int64) ([]exchange.GivenItem, error) {
	gives, err := r.getGives(ctx, []int64{accountID})
	if err != nil {
		return nil, err
	}
	return gives[accountID], nil
}

// FindCandidateAccounts returns accounts whose wishes match the supplied give.
func (r *PostgresRepository) FindCandidateAccounts(ctx context.Context, give exchange.GivenItem, limit int) ([]userCandidate, error) {
	const query = `
		SELECT DISTINCT a.id, a.public_rating, a.shadow_rating
		FROM users AS a
		JOIN wishes AS w ON w.user_id = a.id
		WHERE w.category_id = $1
		  AND w.price_min <= $2
		  AND w.price_max >= $3
		  AND EXISTS (
			SELECT 1 FROM items AS gi
			WHERE gi.user_id = a.id AND gi.status = 'searching'
		  )
		LIMIT $4`

	rows, err := r.pool.Query(ctx, query, give.CategoryID, give.PriceMin, give.PriceMax, limit)
	if err != nil {
		return nil, fmt.Errorf("find candidate accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]userCandidate, 0)
	accountIDs := make([]int64, 0)
	for rows.Next() {
		var account userCandidate
		if err := rows.Scan(&account.ID, &account.PublicRating, &account.ShadowRating); err != nil {
			return nil, fmt.Errorf("scan candidate account: %w", err)
		}
		accounts = append(accounts, account)
		accountIDs = append(accountIDs, account.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidate accounts: %w", err)
	}
	if len(accounts) == 0 {
		return accounts, nil
	}

	wishFilter := &wishQueryFilter{CategoryID: give.CategoryID, PriceMin: give.PriceMin, PriceMax: give.PriceMax}
	wishes, err := r.getWishes(ctx, accountIDs, wishFilter)
	if err != nil {
		return nil, err
	}
	gives, err := r.getGives(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		accounts[i].Wishes = wishes[accounts[i].ID]
		accounts[i].Gives = gives[accounts[i].ID]
	}

	return accounts, nil
}

// ListChains returns every persisted chain with its nested barters.
func (r *PostgresRepository) ListChains(ctx context.Context) ([]exchange.Chain, error) {
	return r.getChains(ctx, "")
}

// CreateChain inserts a chain together with its barters and participants,
// ignoring an already persisted signature.
func (r *PostgresRepository) CreateChain(ctx context.Context, chain exchange.Chain) error {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create chain transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const chainQuery = `
		INSERT INTO chains (root_user_id, root_wish_id, signature, score, status)
		VALUES ($1, $2, $3, $4, 'matched')
		ON CONFLICT (signature) DO NOTHING
		RETURNING id`

	var chainID int64
	err = tx.QueryRow(ctx, chainQuery, chain.RootID, chain.RootWish.ID, chain.Signature, chain.Score).Scan(&chainID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("create chain: %w", err)
	}

	for position, barter := range chain.Barters {
		const barterQuery = `
			INSERT INTO chain_barters (chain_id, from_user_id, to_user_id, give_item_id, wish_id, position)
			VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := tx.Exec(
			ctx,
			barterQuery,
			chainID,
			barter.From,
			barter.To,
			barter.Give.ID,
			barter.Wish.ID,
			position+1,
		); err != nil {
			return fmt.Errorf("create chain barter: %w", err)
		}
	}

	for _, userID := range chainParticipantIDs(chain) {
		const participantQuery = `
			INSERT INTO chain_participants (chain_id, user_id)
			VALUES ($1, $2)`
		if _, err := tx.Exec(ctx, participantQuery, chainID, userID); err != nil {
			return fmt.Errorf("create chain participant: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create chain transaction: %w", err)
	}

	return nil
}

// IDEA I don't really think if we need this function...

func (r *PostgresRepository) UpdateChainStatus(ctx context.Context, newStatus exchange.Status, participant exchange.Participant) (exchange.Chain, error) {

	return exchange.Chain{}, errors.New("not implemented")
}

// GetChainsByItem returns the chains rooted at the given user's wished item.
func (r *PostgresRepository) GetChainsByItem(ctx context.Context, userId int64, wishedItemId int64) ([]exchange.Chain, error) {
	const where = `WHERE root_user_id = $1 AND root_wish_id = $2`
	return r.getChains(ctx, where, userId, wishedItemId)
}

// GetChainsByUser returns the chains where the user is the root or a participant.
func (r *PostgresRepository) GetChainsByUser(ctx context.Context, userId int64) ([]exchange.Chain, error) {
	const where = `WHERE root_user_id = $1
		OR id IN (SELECT chain_id FROM chain_participants WHERE user_id = $1)`
	return r.getChains(ctx, where, userId)
}

// TODO implement this

func (r *PostgresRepository) AcceptChain(ctx context.Context, chainId, userId int64) (exchange.Chain, error) {

	return exchange.Chain{}, errors.New("[cb/repository] Not implemented")
}

// TODO implement this

func (r *PostgresRepository) DeclineChain(ctx context.Context, chainId, userId int64) (exchange.Chain, error) {

	return exchange.Chain{}, errors.New("[cb/repository] Not implemented")
}

// MARK: Private functions

// chainParticipantIDs returns the distinct user IDs taking part in a chain.
func chainParticipantIDs(chain exchange.Chain) []int64 {
	seen := make(map[int64]bool)
	ids := make([]int64, 0)
	add := func(id int64) {
		if id == 0 || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}

	add(chain.RootID)
	for _, barter := range chain.Barters {
		add(barter.From)
		add(barter.To)
	}

	return ids
}

type wishQueryFilter struct {
	CategoryID int64
	PriceMin   int64
	PriceMax   int64
}

func (r *PostgresRepository) getWishes(ctx context.Context, accountIDs []int64, filter *wishQueryFilter) (map[int64][]exchange.WishedItem, error) {
	query := `
		SELECT id, user_id, category_id, price_min, price_max, title_keywords, created_at, updated_at
		FROM wishes
		WHERE user_id = ANY($1)`
	arguments := []any{accountIDs}
	if filter != nil {
		query += ` AND category_id = $2 AND price_min <= $3 AND price_max >= $4`
		arguments = append(arguments, filter.CategoryID, filter.PriceMin, filter.PriceMax)
	}

	rows, err := r.pool.Query(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("get account wishes: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]exchange.WishedItem, len(accountIDs))
	for rows.Next() {
		var wish exchange.WishedItem
		if err := rows.Scan(
			&wish.ID,
			&wish.UserID,
			&wish.CategoryID,
			&wish.PriceMin,
			&wish.PriceMax,
			&wish.TitleKeywords,
			&wish.CreatedAt,
			&wish.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account wish: %w", err)
		}
		result[wish.UserID] = append(result[wish.UserID], wish)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account wishes: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) getWishesByIDs(ctx context.Context, wishIDs []int64) (map[int64]exchange.WishedItem, error) {
	if len(wishIDs) == 0 {
		return map[int64]exchange.WishedItem{}, nil
	}

	const query = `
		SELECT id, user_id, category_id, price_min, price_max, title_keywords, created_at, updated_at
		FROM wishes
		WHERE id = ANY($1)`

	rows, err := r.pool.Query(ctx, query, wishIDs)
	if err != nil {
		return nil, fmt.Errorf("get wishes by ids: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]exchange.WishedItem, len(wishIDs))
	for rows.Next() {
		var wish exchange.WishedItem
		if err := rows.Scan(
			&wish.ID,
			&wish.UserID,
			&wish.CategoryID,
			&wish.PriceMin,
			&wish.PriceMax,
			&wish.TitleKeywords,
			&wish.CreatedAt,
			&wish.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan wish by id: %w", err)
		}
		result[wish.ID] = wish
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wishes by ids: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) getGives(ctx context.Context, accountIDs []int64) (map[int64][]exchange.GivenItem, error) {
	const query = `
		SELECT id, user_id, title, category_id, price_min, price_max, status, created_at, updated_at
		FROM items
		WHERE user_id = ANY($1)
		  AND status = 'searching'
		ORDER BY updated_at`

	rows, err := r.pool.Query(ctx, query, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("get account gives: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]exchange.GivenItem, len(accountIDs))
	for rows.Next() {
		var give exchange.GivenItem
		var status string
		if err := rows.Scan(
			&give.ID,
			&give.UserID,
			&give.Title,
			&give.CategoryID,
			&give.PriceMin,
			&give.PriceMax,
			&status,
			&give.CreatedAt,
			&give.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account give: %w", err)
		}
		give.Status = "searching"
		result[give.UserID] = append(result[give.UserID], give)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account gives: %w", err)
	}

	return result, nil
}

// getChains loads chains matching an optional WHERE clause together with
// their root wishes and nested barters.
func (r *PostgresRepository) getChains(ctx context.Context, where string, args ...any) ([]exchange.Chain, error) {
	query := `
		SELECT id, root_user_id, root_wish_id, signature, status, score, created_at
		FROM chains`
	if where != "" {
		query += " " + where
	}
	query += " ORDER BY id"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list chains: %w", err)
	}
	defer rows.Close()

	chains := make([]exchange.Chain, 0)
	chainIndex := make(map[int64]int)
	for rows.Next() {
		var chain exchange.Chain
		var status string
		var rootWishID int64
		if err := rows.Scan(
			&chain.ID,
			&chain.RootID,
			&rootWishID,
			&chain.Signature,
			&status,
			&chain.Score,
			&chain.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chain: %w", err)
		}
		chain.RootWish.ID = rootWishID
		chain.Status = exchange.Status(status)
		chainIndex[chain.ID] = len(chains)
		chains = append(chains, chain)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chains: %w", err)
	}
	if len(chains) == 0 {
		return chains, nil
	}

	rootWishIDs := make([]int64, 0, len(chains))
	for _, chain := range chains {
		rootWishIDs = append(rootWishIDs, chain.RootWish.ID)
	}

	rootWishes, err := r.getWishesByIDs(ctx, rootWishIDs)
	if err != nil {
		return nil, err
	}
	for i := range chains {
		if wish, ok := rootWishes[chains[i].RootWish.ID]; ok {
			chains[i].RootWish = wish
		}
	}

	chainIDs := make([]int64, 0, len(chains))
	for _, chain := range chains {
		chainIDs = append(chainIDs, chain.ID)
	}

	const barterQuery = `
		SELECT cb.id, cb.chain_id, cb.position, cb.from_user_id, cb.to_user_id,
		       cb.status,
		       cb.give_item_id, gi.user_id, gi.title, gi.category_id, gi.price_min, gi.price_max, gi.status,
		       cb.wish_id, w.user_id, w.category_id, w.price_min, w.price_max, w.title_keywords
		FROM chain_barters AS cb
		JOIN items AS gi ON gi.id = cb.give_item_id
		JOIN wishes AS w ON w.id = cb.wish_id
		WHERE cb.chain_id = ANY($1)
		ORDER BY cb.chain_id, cb.position`

	barterRows, err := r.pool.Query(ctx, barterQuery, chainIDs)
	if err != nil {
		return nil, fmt.Errorf("list chain barters: %w", err)
	}
	defer barterRows.Close()

	for barterRows.Next() {
		var barter exchange.Barter
		var chainID, giveID, wishID, position int64
		var status, giveStatus string
		if err := barterRows.Scan(
			&barter.ID,
			&chainID,
			&position,
			&barter.From,
			&barter.To,
			&status,
			&giveID,
			&barter.Give.UserID,
			&barter.Give.Title,
			&barter.Give.CategoryID,
			&barter.Give.PriceMin,
			&barter.Give.PriceMax,
			&giveStatus,
			&wishID,
			&barter.Wish.UserID,
			&barter.Wish.CategoryID,
			&barter.Wish.PriceMin,
			&barter.Wish.PriceMax,
			&barter.Wish.TitleKeywords,
		); err != nil {
			return nil, fmt.Errorf("scan chain barter: %w", err)
		}
		barter.Give.ID = giveID
		barter.Wish.ID = wishID
		barter.Give.Status = "searching"
		barter.Status = exchange.BarterStatus(status)
		idx, ok := chainIndex[chainID]
		if !ok {
			continue
		}
		chains[idx].Barters = append(chains[idx].Barters, barter)
	}
	if err := barterRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chain barters: %w", err)
	}

	return chains, nil
}
