package matching

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/item"
)

// Matcher describes future cycle discovery without prescribing an implementation.
type Matcher interface {
	FindCycles(
		ctx context.Context,
		startItem item.Item,
		availableItems []item.Item,
		maxDepth int,
	) ([]Cycle, error)
}

// TODO: implement cycle matching with a maximum depth of 2 to 5 participants.
