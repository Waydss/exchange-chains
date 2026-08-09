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

// GetChainByID returns one chain visible to the current participant.
func (s *Service) GetChainByID(ctx context.Context, chainID, userID int64) (exchange.Chain, error) {
	if err := validatePositive("chain_id", chainID); err != nil {
		return exchange.Chain{}, err
	}
	if err := validatePositive("user_id", userID); err != nil {
		return exchange.Chain{}, err
	}

	chain, err := s.repository.GetChainByID(ctx, chainID, userID)
	if err != nil {
		return exchange.Chain{}, fmt.Errorf("get chain by ID: %w", err)
	}

	return chain, nil
}

// AcceptChain records the current participant's acceptance in PostgreSQL.
func (s *Service) AcceptChain(ctx context.Context, chainID, userID int64) (AcceptResult, error) {
	if err := validatePositive("chain_id", chainID); err != nil {
		return AcceptResult{}, err
	}
	if err := validatePositive("user_id", userID); err != nil {
		return AcceptResult{}, err
	}

	result, err := s.repository.AcceptChain(ctx, chainID, userID)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("accept chain in PostgreSQL: %w", err)
	}

	return result, nil
}

// DeclineChain cancels a chain in PostgreSQL and records the rejected item in Redis.
func (s *Service) DeclineChain(ctx context.Context, chainID, userID int64) (DeclineResult, error) {
	if err := validatePositive("chain_id", chainID); err != nil {
		return DeclineResult{}, err
	}
	if err := validatePositive("user_id", userID); err != nil {
		return DeclineResult{}, err
	}
	if s.rejectionStore == nil {
		return DeclineResult{}, errors.New("[cb/service] rejection store is not configured")
	}

	result, err := s.repository.DeclineChain(ctx, chainID, userID)
	if err != nil {
		return DeclineResult{}, fmt.Errorf("decline chain in PostgreSQL: %w", err)
	}

	if err := s.rejectionStore.Reject(ctx, userID, result.ReceivedItemID); err != nil {
		return DeclineResult{}, fmt.Errorf("record rejected received item: %w", err)
	}

	return result, nil
}
