package wish

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/category"
)

// Details contains a wanted item together with its category.
type Details struct {
	Wish     WishedItem
	Category category.Category
}

// Repository contains wanted-item reads required by the API.
type Repository interface {
	ListByUserID(ctx context.Context, userID int64) ([]Details, error)
}
