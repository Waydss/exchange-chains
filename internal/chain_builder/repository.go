package cb

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/chain_builder/entities"
)

// Repository contains the database operations required by the chain builder.
type Repository interface {
	GetAccount(ctx context.Context, accountID uint) (entities.Account, error)
	FindCandidateAccounts(ctx context.Context, give entities.GivedItem, limit int) ([]entities.Account, error)
	CreateChain(ctx context.Context, chain entities.Chain) error
}
