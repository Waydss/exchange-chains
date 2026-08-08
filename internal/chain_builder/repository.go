package cb

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

// Repository contains the database operations required by the chain builder.
type Repository interface {

	// Used internally for getting all the necessary information about the Users and their accounts
	GetAccount(ctx context.Context, accountID int64) (userCandidate, error)

	// Used internally for getting all possible account's Given Items
	GetAccountGives(ctx context.Context, accountID int64) ([]exchange.GivenItem, error)

	// Used internally for getting all possible account's Given Items
	FindCandidateAccounts(ctx context.Context, give exchange.GivenItem, limit int) ([]userCandidate, error)

	// Internal function, just for tests
	CreateChain(ctx context.Context, chain exchange.Chain) error

	// Internal function, just for tests
	ListChains(ctx context.Context) ([]exchange.Chain, error)

	GetChainsByItem(ctx context.Context, userId int64, wishedItemId int64) ([]exchange.Chain, error)
	GetChainsByUser(ctx context.Context, userId int64) ([]exchange.Chain, error)
	AcceptChain(ctx context.Context, chainId, userId int64) (exchange.Chain, error)
	DeclineChain(ctx context.Context, chainId, userId int64) (exchange.Chain, error)

	// IDEA I don't really think if we need this function...
	UpdateChainStatus(ctx context.Context, newStatus exchange.Status, participant exchange.Participant) (exchange.Chain, error)
}
