package rejection

import (
	"context"
	"fmt"
)

type rejectionWriter interface {
	Reject(ctx context.Context, userID, itemID int64) error
}

// DeclineChainUseCase records the item that caused a user to decline a chain.
type DeclineChainUseCase struct {
	store rejectionWriter
}

// NewDeclineChainUseCase creates the chain-decline use case.
func NewDeclineChainUseCase(store rejectionWriter) *DeclineChainUseCase {
	return &DeclineChainUseCase{store: store}
}

// Execute records a temporary rejection without changing PostgreSQL chain state.
func (uc *DeclineChainUseCase) Execute(ctx context.Context, userID, chainID, receivedItemID int64) error {
	if err := validatePositive("user_id", userID); err != nil {
		return err
	}
	if err := validatePositive("chain_id", chainID); err != nil {
		return err
	}
	if err := validatePositive("received_item_id", receivedItemID); err != nil {
		return err
	}

	// TODO: After chain storage integration, cancel the selected chain and all
	// pending chains where this user is expected to receive the rejected item.
	if err := uc.store.Reject(ctx, userID, receivedItemID); err != nil {
		return fmt.Errorf("record rejected received item: %w", err)
	}

	return nil
}
