package cb

import (
	"context"
	"errors"
	"fmt"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

const chain_request_queue_size = 100

// Service owns chain operations and their PostgreSQL and Redis dependencies.
type Service struct {
	repository     Repository
	rejectionStore RejectionWriter
	queue          chan ChainRequest
	userRequester  UserRequester
}

type UserRequester interface {
	// Returns the publicRating and shadowRating for the user, found by provided UserID. These will be used in construction for the userCandidate in chain_builder package
	GetUserCandidateBase(ctx context.Context, userId int64) (publicRating float64, shadowRating int, err error)
}

// NewService creates a chain-building service.
func NewService(repository Repository, userService UserRequester, rejectionStore RejectionWriter) *Service {
	return &Service{
		repository:     repository,
		rejectionStore: rejectionStore,
		queue:          make(chan ChainRequest, chain_request_queue_size),
		userRequester:  userService,
	}
}

// EnqueueChainRequest adds a request to the asynchronous chain-building queue.
func (s *Service) EnqueueChainRequest(userID, wishedItemID int64) {
	s.queue <- ChainRequest{UserID: userID, WishedItemId: wishedItemID}
}

func (s *Service) GetChainsByUser(ctx context.Context, userId int64) ([]exchange.Chain, error) {
	return s.repository.GetChainsByUser(ctx, userId)
}

func (s *Service) GetChainsByItem(ctx context.Context, userId, wishedItemID int64) ([]exchange.Chain, error) {
	return s.repository.GetChainsByItem(ctx, userId, wishedItemID)
}

// TODO implement this

func (s *Service) AcceptChain(ctx context.Context, chainId, userId int64) (exchange.Chain, error) {

	return exchange.Chain{}, errors.New("[cb/service] Not implemented")
}

// DeclineChain cancels a chain in PostgreSQL and records the rejected item in Redis.
func (s *Service) DeclineChain(ctx context.Context, chainID, userID, receivedItemID int64) error {
	if err := validatePositive("chain_id", chainID); err != nil {
		return err
	}
	if err := validatePositive("user_id", userID); err != nil {
		return err
	}
	if err := validatePositive("received_item_id", receivedItemID); err != nil {
		return err
	}
	if s.rejectionStore == nil {
		return errors.New("[cb/service] rejection store is not configured")
	}
	if _, err := s.repository.DeclineChain(ctx, chainID, userID); err != nil {
		return fmt.Errorf("decline chain in PostgreSQL: %w", err)
	}

	if err := s.rejectionStore.Reject(ctx, userID, receivedItemID); err != nil {
		return fmt.Errorf("record rejected received item: %w", err)
	}

	return nil
}
