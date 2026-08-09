package simulation

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/category"
	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/L-ka080/DDB-6-case/internal/user"
	"github.com/L-ka080/DDB-6-case/internal/wish"
)

// Repository abstracts the bulk persistence needed for seeding.
type Repository interface {
	// LoadCategories returns every catalogue category.
	LoadCategories(ctx context.Context) ([]category.Category, error)
	// LoadLastIDs returns the current maximum primary keys of users,
	// items, and wishes so generated rows never collide.
	LoadLastIDs(ctx context.Context) (userID, itemID, wishID int64, err error)
	// BulkInsertUsers copies many users in one round trip.
	BulkInsertUsers(ctx context.Context, users []user.User) error
	// BulkInsertItems copies many items in one round trip.
	BulkInsertItems(ctx context.Context, items []item.Item) error
	// BulkInsertWishes copies many wishes in one round trip.
	BulkInsertWishes(ctx context.Context, wishes []wish.WishedItem) error
	// SyncSequences advances the serial sequences past the copied ids.
	SyncSequences(ctx context.Context) error
	// ClearData removes all simulated rows and their dependents.
	ClearData(ctx context.Context) error
}
