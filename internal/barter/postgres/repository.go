package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/barter"
	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cancelReasonReturnedAfterRejection = "returned_after_rejection"

type Repository struct {
	pool *pgxpool.Pool
}

var _ barter.Repository = (*Repository)(nil)

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SelectTransferMethod(
	ctx context.Context,
	input barter.SelectTransferMethodInput,
) (barter.OperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return barter.OperationResult{}, fmt.Errorf("begin select transfer method transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockState(ctx, tx, input.ChainID, input.BarterID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if !isBarterParticipant(state, input.UserID) {
		return barter.OperationResult{}, barter.ErrNotParticipant
	}
	if !allowsForwardProgress(state.ChainStatus) {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrChainStatusConflict, state.ChainStatus)
	}
	if state.BarterStatus != exchange.BarterPending && state.BarterStatus != exchange.BarterMethodSelected {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrBarterStatusConflict, state.BarterStatus)
	}

	const query = `
		UPDATE chain_barters
		SET status = $1, transfer_method = $2
		WHERE id = $3`
	if _, err := tx.Exec(ctx, query, exchange.BarterMethodSelected, input.Method, input.BarterID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("update transfer method: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return barter.OperationResult{}, fmt.Errorf("commit select transfer method transaction: %w", err)
	}

	method := input.Method
	return barter.OperationResult{
		ChainID:        input.ChainID,
		BarterID:       input.BarterID,
		BarterStatus:   exchange.BarterMethodSelected,
		ChainStatus:    state.ChainStatus,
		TransferMethod: &method,
	}, nil
}

func (r *Repository) MarkGiven(
	ctx context.Context,
	input barter.OperationInput,
) (barter.OperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return barter.OperationResult{}, fmt.Errorf("begin mark given transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockState(ctx, tx, input.ChainID, input.BarterID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if state.FromUserID != input.UserID {
		return barter.OperationResult{}, barter.ErrNotParticipant
	}
	if !allowsForwardProgress(state.ChainStatus) {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrChainStatusConflict, state.ChainStatus)
	}
	if state.BarterStatus != exchange.BarterMethodSelected {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrBarterStatusConflict, state.BarterStatus)
	}

	if err := updateItemStatus(ctx, tx, state.GiveItemID, item.StatusReserved, item.StatusInTransit); err != nil {
		return barter.OperationResult{}, err
	}

	const barterQuery = `
		UPDATE chain_barters
		SET status = $1, given_at = NOW()
		WHERE id = $2`
	if _, err := tx.Exec(ctx, barterQuery, exchange.BarterGiven, input.BarterID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("mark barter given: %w", err)
	}

	chainStatus := state.ChainStatus
	if chainStatus == exchange.StatusLocked {
		const chainQuery = `UPDATE chains SET status = $1 WHERE id = $2`
		if _, err := tx.Exec(ctx, chainQuery, exchange.StatusInDelivery, input.ChainID); err != nil {
			return barter.OperationResult{}, fmt.Errorf("mark chain in delivery: %w", err)
		}
		chainStatus = exchange.StatusInDelivery
	}

	if err := tx.Commit(ctx); err != nil {
		return barter.OperationResult{}, fmt.Errorf("commit mark given transaction: %w", err)
	}

	return barter.OperationResult{
		ChainID:        input.ChainID,
		BarterID:       input.BarterID,
		BarterStatus:   exchange.BarterGiven,
		ChainStatus:    chainStatus,
		TransferMethod: state.TransferMethod,
	}, nil
}

func (r *Repository) ConfirmReceived(
	ctx context.Context,
	input barter.OperationInput,
) (barter.OperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return barter.OperationResult{}, fmt.Errorf("begin confirm received transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockState(ctx, tx, input.ChainID, input.BarterID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if state.ToUserID != input.UserID {
		return barter.OperationResult{}, barter.ErrNotParticipant
	}
	if state.ChainStatus != exchange.StatusInDelivery {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrChainStatusConflict, state.ChainStatus)
	}
	if state.BarterStatus != exchange.BarterGiven {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrBarterStatusConflict, state.BarterStatus)
	}

	const barterQuery = `
		UPDATE chain_barters
		SET status = $1, received_at = NOW()
		WHERE id = $2`
	if _, err := tx.Exec(ctx, barterQuery, exchange.BarterReceived, input.BarterID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("confirm barter received: %w", err)
	}

	chainStatus := exchange.StatusInDelivery
	complete, err := allBartersHaveStatus(ctx, tx, input.ChainID, exchange.BarterReceived)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if complete {
		if err := exchangeChainItems(ctx, tx, input.ChainID); err != nil {
			return barter.OperationResult{}, err
		}
		const chainQuery = `
			UPDATE chains
			SET status = $1, completed_at = NOW()
			WHERE id = $2`
		if _, err := tx.Exec(ctx, chainQuery, exchange.StatusCompleted, input.ChainID); err != nil {
			return barter.OperationResult{}, fmt.Errorf("complete chain: %w", err)
		}
		chainStatus = exchange.StatusCompleted
	}

	if err := tx.Commit(ctx); err != nil {
		return barter.OperationResult{}, fmt.Errorf("commit confirm received transaction: %w", err)
	}

	return barter.OperationResult{
		ChainID:        input.ChainID,
		BarterID:       input.BarterID,
		BarterStatus:   exchange.BarterReceived,
		ChainStatus:    chainStatus,
		TransferMethod: state.TransferMethod,
	}, nil
}

func (r *Repository) Reject(ctx context.Context, input barter.RejectInput) (barter.OperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return barter.OperationResult{}, fmt.Errorf("begin reject transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockState(ctx, tx, input.ChainID, input.BarterID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if state.ToUserID != input.UserID {
		return barter.OperationResult{}, barter.ErrNotParticipant
	}
	if state.ChainStatus != exchange.StatusInDelivery {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrChainStatusConflict, state.ChainStatus)
	}
	if state.BarterStatus != exchange.BarterGiven {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrBarterStatusConflict, state.BarterStatus)
	}

	const barterQuery = `
		UPDATE chain_barters
		SET status = $1, rejected_at = NOW(), rejection_reason = $2
		WHERE id = $3`
	if _, err := tx.Exec(ctx, barterQuery, exchange.BarterRejected, input.Reason, input.BarterID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("reject barter: %w", err)
	}

	const chainQuery = `UPDATE chains SET status = $1 WHERE id = $2`
	if _, err := tx.Exec(ctx, chainQuery, exchange.StatusReturning, input.ChainID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("mark chain returning: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return barter.OperationResult{}, fmt.Errorf("commit reject transaction: %w", err)
	}

	return barter.OperationResult{
		ChainID:        input.ChainID,
		BarterID:       input.BarterID,
		BarterStatus:   exchange.BarterRejected,
		ChainStatus:    exchange.StatusReturning,
		TransferMethod: state.TransferMethod,
	}, nil
}

func (r *Repository) MarkReturnStarted(
	ctx context.Context,
	input barter.OperationInput,
) (barter.OperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return barter.OperationResult{}, fmt.Errorf("begin mark return started transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockState(ctx, tx, input.ChainID, input.BarterID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if state.ToUserID != input.UserID {
		return barter.OperationResult{}, barter.ErrNotParticipant
	}
	if state.ChainStatus != exchange.StatusReturning {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrChainStatusConflict, state.ChainStatus)
	}
	if !requiresReturn(state.BarterStatus) {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrBarterStatusConflict, state.BarterStatus)
	}

	const query = `
		UPDATE chain_barters
		SET status = $1, return_started_at = NOW()
		WHERE id = $2`
	if _, err := tx.Exec(ctx, query, exchange.BarterReturning, input.BarterID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("mark barter returning: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return barter.OperationResult{}, fmt.Errorf("commit mark return started transaction: %w", err)
	}

	return barter.OperationResult{
		ChainID:        input.ChainID,
		BarterID:       input.BarterID,
		BarterStatus:   exchange.BarterReturning,
		ChainStatus:    exchange.StatusReturning,
		TransferMethod: state.TransferMethod,
	}, nil
}

func (r *Repository) ConfirmReturned(
	ctx context.Context,
	input barter.OperationInput,
) (barter.OperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return barter.OperationResult{}, fmt.Errorf("begin confirm returned transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	state, err := lockState(ctx, tx, input.ChainID, input.BarterID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if state.FromUserID != input.UserID {
		return barter.OperationResult{}, barter.ErrNotParticipant
	}
	if state.ChainStatus != exchange.StatusReturning {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrChainStatusConflict, state.ChainStatus)
	}
	if state.BarterStatus != exchange.BarterReturning {
		return barter.OperationResult{}, fmt.Errorf("%w: %s", barter.ErrBarterStatusConflict, state.BarterStatus)
	}

	if err := updateItemStatus(ctx, tx, state.GiveItemID, item.StatusInTransit, item.StatusReserved); err != nil {
		return barter.OperationResult{}, err
	}

	const barterQuery = `
		UPDATE chain_barters
		SET status = $1, returned_at = NOW()
		WHERE id = $2`
	if _, err := tx.Exec(ctx, barterQuery, exchange.BarterReturned, input.BarterID); err != nil {
		return barter.OperationResult{}, fmt.Errorf("confirm barter returned: %w", err)
	}

	chainStatus := exchange.StatusReturning
	hasOpenReturns, err := chainHasOpenReturns(ctx, tx, input.ChainID)
	if err != nil {
		return barter.OperationResult{}, err
	}
	if !hasOpenReturns {
		if err := releaseChainItems(ctx, tx, input.ChainID); err != nil {
			return barter.OperationResult{}, err
		}
		const chainQuery = `
			UPDATE chains
			SET status = $1, cancelled_at = NOW(), cancel_reason = $2
			WHERE id = $3`
		if _, err := tx.Exec(
			ctx,
			chainQuery,
			exchange.StatusCancelled,
			cancelReasonReturnedAfterRejection,
			input.ChainID,
		); err != nil {
			return barter.OperationResult{}, fmt.Errorf("cancel returned chain: %w", err)
		}
		chainStatus = exchange.StatusCancelled
	}

	if err := tx.Commit(ctx); err != nil {
		return barter.OperationResult{}, fmt.Errorf("commit confirm returned transaction: %w", err)
	}

	return barter.OperationResult{
		ChainID:        input.ChainID,
		BarterID:       input.BarterID,
		BarterStatus:   exchange.BarterReturned,
		ChainStatus:    chainStatus,
		TransferMethod: state.TransferMethod,
	}, nil
}

type lockedState struct {
	ChainStatus    exchange.Status
	BarterStatus   exchange.BarterStatus
	TransferMethod *exchange.TransferMethod
	FromUserID     int64
	ToUserID       int64
	GiveItemID     int64
}

func lockState(ctx context.Context, tx pgx.Tx, chainID, barterID int64) (lockedState, error) {
	const chainQuery = `SELECT status FROM chains WHERE id = $1 FOR UPDATE`

	var state lockedState
	if err := tx.QueryRow(ctx, chainQuery, chainID).Scan(&state.ChainStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return lockedState{}, barter.ErrNotFound
		}
		return lockedState{}, fmt.Errorf("lock chain: %w", err)
	}

	const barterQuery = `
		SELECT status, transfer_method, from_user_id, to_user_id, give_item_id
		FROM chain_barters
		WHERE id = $1 AND chain_id = $2
		FOR UPDATE`
	var transferMethod pgtype.Text
	if err := tx.QueryRow(ctx, barterQuery, barterID, chainID).Scan(
		&state.BarterStatus,
		&transferMethod,
		&state.FromUserID,
		&state.ToUserID,
		&state.GiveItemID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return lockedState{}, barter.ErrNotFound
		}
		return lockedState{}, fmt.Errorf("lock barter: %w", err)
	}
	if transferMethod.Valid {
		method := exchange.TransferMethod(transferMethod.String)
		state.TransferMethod = &method
	}

	return state, nil
}

func isBarterParticipant(state lockedState, userID int64) bool {
	return state.FromUserID == userID || state.ToUserID == userID
}

func allowsForwardProgress(status exchange.Status) bool {
	return status == exchange.StatusLocked || status == exchange.StatusInDelivery
}

func requiresReturn(status exchange.BarterStatus) bool {
	switch status {
	case exchange.BarterGiven, exchange.BarterReceived, exchange.BarterRejected:
		return true
	default:
		return false
	}
}

func updateItemStatus(ctx context.Context, tx pgx.Tx, itemID int64, from, to item.Status) error {
	const query = `
		UPDATE items
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3`
	result, err := tx.Exec(ctx, query, to, itemID, from)
	if err != nil {
		return fmt.Errorf("update item status: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("%w: item %d", barter.ErrChainItemsConflict, itemID)
	}

	return nil
}

func allBartersHaveStatus(ctx context.Context, tx pgx.Tx, chainID int64, status exchange.BarterStatus) (bool, error) {
	const query = `
		SELECT NOT EXISTS (
			SELECT 1
			FROM chain_barters
			WHERE chain_id = $1 AND status <> $2
		)`

	var complete bool
	if err := tx.QueryRow(ctx, query, chainID, status).Scan(&complete); err != nil {
		return false, fmt.Errorf("check chain barter statuses: %w", err)
	}

	return complete, nil
}

func chainHasOpenReturns(ctx context.Context, tx pgx.Tx, chainID int64) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM chain_barters
			WHERE chain_id = $1 AND status IN ($2, $3, $4, $5)
		)`

	var exists bool
	if err := tx.QueryRow(
		ctx,
		query,
		chainID,
		exchange.BarterGiven,
		exchange.BarterReceived,
		exchange.BarterRejected,
		exchange.BarterReturning,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check open returns: %w", err)
	}

	return exists, nil
}

func exchangeChainItems(ctx context.Context, tx pgx.Tx, chainID int64) error {
	itemCount, err := chainItemCount(ctx, tx, chainID)
	if err != nil {
		return err
	}

	const query = `
		UPDATE items
		SET status = $1, updated_at = NOW()
		WHERE id IN (
			SELECT give_item_id
			FROM chain_barters
			WHERE chain_id = $2
		)
		  AND status = $3`
	result, err := tx.Exec(ctx, query, item.StatusExchanged, chainID, item.StatusInTransit)
	if err != nil {
		return fmt.Errorf("exchange chain items: %w", err)
	}
	if result.RowsAffected() != itemCount {
		return fmt.Errorf("%w: chain %d", barter.ErrChainItemsConflict, chainID)
	}

	return nil
}

func releaseChainItems(ctx context.Context, tx pgx.Tx, chainID int64) error {
	itemCount, err := chainItemCount(ctx, tx, chainID)
	if err != nil {
		return err
	}

	const query = `
		UPDATE items
		SET status = $1, updated_at = NOW()
		WHERE id IN (
			SELECT give_item_id
			FROM chain_barters
			WHERE chain_id = $2
		)
		  AND status IN ($3, $4)`
	result, err := tx.Exec(ctx, query, item.StatusSearching, chainID, item.StatusReserved, item.StatusInTransit)
	if err != nil {
		return fmt.Errorf("release chain items: %w", err)
	}
	if result.RowsAffected() != itemCount {
		return fmt.Errorf("%w: chain %d", barter.ErrChainItemsConflict, chainID)
	}

	return nil
}

func chainItemCount(ctx context.Context, tx pgx.Tx, chainID int64) (int64, error) {
	const query = `SELECT COUNT(DISTINCT give_item_id) FROM chain_barters WHERE chain_id = $1`

	var count int64
	if err := tx.QueryRow(ctx, query, chainID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count chain items: %w", err)
	}
	if count == 0 {
		return 0, fmt.Errorf("%w: chain %d has no items", barter.ErrChainItemsConflict, chainID)
	}

	return count, nil
}
