package cb

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

// Repository contains the database operations required by the chain builder.
type Repository interface {
	GetAccount(ctx context.Context, accountID int64) (userCandidate, error)
	GetAccountGives(ctx context.Context, accountID int64) ([]exchange.GivenItem, error)
	FindCandidateAccounts(ctx context.Context, give exchange.GivenItem, limit int) ([]userCandidate, error)
	CreateChain(ctx context.Context, chain exchange.Chain) error
	ListChains(ctx context.Context) ([]exchange.Chain, error)
}
