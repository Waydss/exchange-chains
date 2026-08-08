package cb

import (
	"context"
	"errors"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

const chain_request_queue_size = 100

// Service owns the chain-building queue and its database dependency.
type Service struct {
	repository    Repository
	queue         chan ChainRequest
	userRequester UserRequester
}

type UserRequester interface {
	// Returns the publicRating and shadowRating for the user, found by provided UserID. These will be used in construction for the userCandidate in chain_builder package
	GetUserCandidateBase(ctx context.Context, userId int64) (publicRating float64, shadowRating int, err error)
}

// NewService creates a chain-building service.
func NewService(repository Repository, userService UserRequester) *Service {
	return &Service{
		repository:    repository,
		queue:         make(chan ChainRequest, chain_request_queue_size),
		userRequester: userService,
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

// TODO implement this

func (s *Service) DeclineChain(ctx context.Context, chainId, userId int64) (exchange.Chain, error) {

	return exchange.Chain{}, errors.New("[cb/service] Not implemented")
}
