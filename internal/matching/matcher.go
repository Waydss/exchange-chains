package matching

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/L-ka080/DDB-6-case/internal/item"
)

// Matcher describes future cycle discovery without prescribing an implementation.
type Matcher interface {
	FindCycles(
		ctx context.Context,
		startItem item.Item,
		availableItems []item.Item,
		maxDepth int,
		// I changed it so the linter would shut up. I don't remember why do we have this interface
	) ([]exchange.Chain, error)
}

// TODO: implement cycle matching with a maximum depth of 2 to 5 participants.
