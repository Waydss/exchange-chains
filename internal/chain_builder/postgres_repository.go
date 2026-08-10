package cb

import (
	"context"
	"errors"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cancelReasonParticipantDeclined = "participant_declined"

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

func (r *PostgresRepository) FillRootCandidate(ctx context.Context, candidateBase *userCandidate) error {
	wishes, err := r.getWishes(ctx, []int64{candidateBase.ID}, nil)
	if err != nil {
		return err
	}
	gives, err := r.getGives(ctx, []int64{candidateBase.ID})
	if err != nil {
		return err
	}

	candidateBase.Wishes = wishes[candidateBase.ID]
	candidateBase.Gives = gives[candidateBase.ID]

	return nil
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
			WHERE gi.user_id = a.id AND gi.status = $4
		  )
		LIMIT $5`

	rows, err := r.pool.Query(
		ctx,
		query,
		give.CategoryID,
		give.PriceMin,
		give.PriceMax,
		item.StatusSearching,
		limit,
	)
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
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (signature) DO NOTHING
		RETURNING id`

	var chainID int64
	err = tx.QueryRow(
		ctx,
		chainQuery,
		chain.RootID,
		chain.RootWish.ID,
		chain.Signature,
		chain.Score,
		exchange.StatusMatched,
	).Scan(&chainID)
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

// GetChainByID returns one chain after verifying that the user participates in it.
func (r *PostgresRepository) GetChainByID(ctx context.Context, chainID, userID int64) (exchange.Chain, error) {
	const accessQuery = `
		SELECT
			EXISTS (SELECT 1 FROM chains WHERE id = $1),
			EXISTS (
				SELECT 1
				FROM chain_participants
				WHERE chain_id = $1 AND user_id = $2
			)`

	var chainExists, isParticipant bool
	if err := r.pool.QueryRow(ctx, accessQuery, chainID, userID).Scan(&chainExists, &isParticipant); err != nil {
		return exchange.Chain{}, fmt.Errorf("check chain access: %w", err)
	}
	if !chainExists {
		return exchange.Chain{}, fmt.Errorf("%w: chain %d", ErrChainNotFound, chainID)
	}
	if !isParticipant {
		return exchange.Chain{}, fmt.Errorf("%w: chain %d", ErrNotChainParticipant, chainID)
	}

	chains, err := r.getChains(ctx, "WHERE id = $1", chainID)
	if err != nil {
		return exchange.Chain{}, err
	}
	if len(chains) == 0 {
		return exchange.Chain{}, fmt.Errorf("%w: chain %d", ErrChainNotFound, chainID)
	}

	return chains[0], nil
}

// AcceptChain records one participant's acceptance and locks the chain once
// every participant has accepted it.
func (r *PostgresRepository) AcceptChain(ctx context.Context, chainID, userID int64) (AcceptResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("begin accept chain transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	chainStatus, err := chainStatusForUpdate(ctx, tx, chainID)
	if err != nil {
		return AcceptResult{}, err
	}

	participantStatus, err := participantStatusForUpdate(ctx, tx, chainID, userID)
	if err != nil {
		return AcceptResult{}, err
	}

	if chainStatus == exchange.StatusLocked && participantStatus == exchange.ParticipantAccepted {
		return AcceptResult{
			ChainID:           chainID,
			ParticipantStatus: exchange.ParticipantAccepted,
			ChainStatus:       exchange.StatusLocked,
		}, nil
	}
	if !chainDecisionAllowed(chainStatus) {
		return AcceptResult{}, fmt.Errorf("%w: current status is %s", ErrChainStatusConflict, chainStatus)
	}
	if participantStatus != exchange.ParticipantPending && participantStatus != exchange.ParticipantAccepted {
		return AcceptResult{}, fmt.Errorf(
			"%w: participant status is %s",
			ErrChainStatusConflict,
			participantStatus,
		)
	}

	if participantStatus == exchange.ParticipantPending {
		const acceptParticipantQuery = `
			UPDATE chain_participants
			SET status = $1,
			    accepted_at = NOW(),
			    updated_at = NOW()
			WHERE chain_id = $2 AND user_id = $3`

		if _, err := tx.Exec(ctx, acceptParticipantQuery, exchange.ParticipantAccepted, chainID, userID); err != nil {
			return AcceptResult{}, fmt.Errorf("accept chain participant: %w", err)
		}
	}

	const remainingParticipantsQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM chain_participants
			WHERE chain_id = $1 AND status <> $2
		)`

	var hasRemainingParticipants bool
	if err := tx.QueryRow(
		ctx,
		remainingParticipantsQuery,
		chainID,
		exchange.ParticipantAccepted,
	).Scan(&hasRemainingParticipants); err != nil {
		return AcceptResult{}, fmt.Errorf("check remaining chain participants: %w", err)
	}

	newChainStatus := exchange.StatusPending
	if hasRemainingParticipants {
		const markPendingQuery = `UPDATE chains SET status = $1 WHERE id = $2`
		if _, err := tx.Exec(ctx, markPendingQuery, exchange.StatusPending, chainID); err != nil {
			return AcceptResult{}, fmt.Errorf("mark chain pending: %w", err)
		}
	} else {
		if err := reserveChainItems(ctx, tx, chainID); err != nil {
			return AcceptResult{}, err
		}

		const lockChainQuery = `
			UPDATE chains
			SET status = $1, locked_at = NOW()
			WHERE id = $2`
		if _, err := tx.Exec(ctx, lockChainQuery, exchange.StatusLocked, chainID); err != nil {
			return AcceptResult{}, fmt.Errorf("lock accepted chain: %w", err)
		}
		newChainStatus = exchange.StatusLocked
	}

	if err := tx.Commit(ctx); err != nil {
		return AcceptResult{}, fmt.Errorf("commit accept chain transaction: %w", err)
	}

	return AcceptResult{
		ChainID:           chainID,
		ParticipantStatus: exchange.ParticipantAccepted,
		ChainStatus:       newChainStatus,
	}, nil
}

// DeclineChain marks the current participant as declined, cancels the chain,
// and returns the item that the participant would have received.
func (r *PostgresRepository) DeclineChain(ctx context.Context, chainID, userID int64) (DeclineResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DeclineResult{}, fmt.Errorf("begin decline chain transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	chainStatus, err := chainStatusForUpdate(ctx, tx, chainID)
	if err != nil {
		return DeclineResult{}, err
	}
	if _, err := participantStatusForUpdate(ctx, tx, chainID, userID); err != nil {
		return DeclineResult{}, err
	}
	if !chainDecisionAllowed(chainStatus) {
		return DeclineResult{}, fmt.Errorf("%w: current status is %s", ErrChainStatusConflict, chainStatus)
	}

	const receivedItemQuery = `
		SELECT give_item_id
		FROM chain_barters
		WHERE chain_id = $1 AND to_user_id = $2
		ORDER BY position
		LIMIT 1`

	var receivedItemID int64
	if err := tx.QueryRow(ctx, receivedItemQuery, chainID, userID).Scan(&receivedItemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DeclineResult{}, fmt.Errorf(
				"find received item for chain %d and user %d: no matching barter",
				chainID,
				userID,
			)
		}
		return DeclineResult{}, fmt.Errorf("find received item for declined chain: %w", err)
	}

	const participantQuery = `
		UPDATE chain_participants
		SET status = $1,
		    declined_at = NOW(),
		    updated_at = NOW()
		WHERE chain_id = $2 AND user_id = $3`

	result, err := tx.Exec(ctx, participantQuery, exchange.ParticipantDeclined, chainID, userID)
	if err != nil {
		return DeclineResult{}, fmt.Errorf("decline chain participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return DeclineResult{}, fmt.Errorf("%w: chain %d", ErrNotChainParticipant, chainID)
	}

	const chainQuery = `
		UPDATE chains
		SET status = $1,
		    cancelled_at = NOW(),
		    cancelled_by_user_id = $2,
		    cancel_reason = $3
		WHERE id = $4`

	if _, err := tx.Exec(
		ctx,
		chainQuery,
		exchange.StatusCancelled,
		userID,
		cancelReasonParticipantDeclined,
		chainID,
	); err != nil {
		return DeclineResult{}, fmt.Errorf("cancel chain: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DeclineResult{}, fmt.Errorf("commit decline chain transaction: %w", err)
	}

	return DeclineResult{
		ChainID:        chainID,
		ReceivedItemID: receivedItemID,
		ChainStatus:    exchange.StatusCancelled,
	}, nil
}

func chainStatusForUpdate(ctx context.Context, tx pgx.Tx, chainID int64) (exchange.Status, error) {
	const query = `SELECT status FROM chains WHERE id = $1 FOR UPDATE`

	var status exchange.Status
	if err := tx.QueryRow(ctx, query, chainID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%w: chain %d", ErrChainNotFound, chainID)
		}
		return "", fmt.Errorf("load chain status: %w", err)
	}

	return status, nil
}

func participantStatusForUpdate(ctx context.Context, tx pgx.Tx, chainID, userID int64) (exchange.ParticipantStatus, error) {
	const query = `
		SELECT status
		FROM chain_participants
		WHERE chain_id = $1 AND user_id = $2
		FOR UPDATE`

	var status exchange.ParticipantStatus
	if err := tx.QueryRow(ctx, query, chainID, userID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%w: chain %d", ErrNotChainParticipant, chainID)
		}
		return "", fmt.Errorf("load chain participant status: %w", err)
	}

	return status, nil
}

func chainDecisionAllowed(status exchange.Status) bool {
	return status == exchange.StatusMatched || status == exchange.StatusPending
}

func reserveChainItems(ctx context.Context, tx pgx.Tx, chainID int64) error {
	const countQuery = `
		SELECT COUNT(DISTINCT give_item_id)
		FROM chain_barters
		WHERE chain_id = $1`

	var itemCount int64
	if err := tx.QueryRow(ctx, countQuery, chainID).Scan(&itemCount); err != nil {
		return fmt.Errorf("count chain items: %w", err)
	}
	if itemCount == 0 {
		return errors.New("lock chain: chain has no items")
	}

	const reserveQuery = `
		UPDATE items
		SET status = $1, updated_at = NOW()
		WHERE id IN (
			SELECT give_item_id
			FROM chain_barters
			WHERE chain_id = $2
		)
		  AND status = $3`

	result, err := tx.Exec(ctx, reserveQuery, item.StatusReserved, chainID, item.StatusSearching)
	if err != nil {
		return fmt.Errorf("reserve chain items: %w", err)
	}
	if result.RowsAffected() != itemCount {
		return fmt.Errorf("%w: chain %d", ErrChainItemsUnavailable, chainID)
	}

	return nil
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
		  AND status = $2
		ORDER BY updated_at`

	rows, err := r.pool.Query(ctx, query, accountIDs, item.StatusSearching)
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
		give.Status = item.Status(status)
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
		SELECT id, root_user_id, root_wish_id, signature, status, score,
		       expires_at, locked_at, completed_at, cancelled_at,
		       cancelled_by_user_id, cancel_reason, created_at
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
			&chain.ExpiresAt,
			&chain.LockedAt,
			&chain.CompletedAt,
			&chain.CancelledAt,
			&chain.CancelledByUserID,
			&chain.CancelReason,
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
		       cb.status, cb.transfer_method, cb.given_at, cb.received_at,
		       cb.rejected_at, cb.return_started_at, cb.returned_at, cb.rejection_reason,
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
		var transferMethod pgtype.Text
		if err := barterRows.Scan(
			&barter.ID,
			&chainID,
			&position,
			&barter.From,
			&barter.To,
			&status,
			&transferMethod,
			&barter.GivenAt,
			&barter.ReceivedAt,
			&barter.RejectedAt,
			&barter.ReturnStartedAt,
			&barter.ReturnedAt,
			&barter.RejectionReason,
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
		barter.Give.Status = item.Status(giveStatus)
		barter.Status = exchange.BarterStatus(status)
		if transferMethod.Valid {
			method := exchange.TransferMethod(transferMethod.String)
			barter.TransferMethod = &method
		}
		idx, ok := chainIndex[chainID]
		if !ok {
			continue
		}
		chains[idx].Barters = append(chains[idx].Barters, barter)
	}
	if err := barterRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chain barters: %w", err)
	}
	barterRows.Close()

	const participantQuery = `
		SELECT chain_id, user_id, status, accepted_at, declined_at, created_at, updated_at
		FROM chain_participants
		WHERE chain_id = ANY($1)
		ORDER BY chain_id, user_id`

	participantRows, err := r.pool.Query(ctx, participantQuery, chainIDs)
	if err != nil {
		return nil, fmt.Errorf("list chain participants: %w", err)
	}
	defer participantRows.Close()

	for participantRows.Next() {
		var participant exchange.ChainParticipant
		if err := participantRows.Scan(
			&participant.ChainID,
			&participant.UserID,
			&participant.Status,
			&participant.AcceptedAt,
			&participant.DeclinedAt,
			&participant.CreatedAt,
			&participant.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chain participant: %w", err)
		}

		index, ok := chainIndex[participant.ChainID]
		if !ok {
			continue
		}
		chains[index].Participants = append(chains[index].Participants, participant)
	}
	if err := participantRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chain participants: %w", err)
	}

	return chains, nil
}
