package item

import "context"

// ItemRepository describes the persistence operation needed by future matching.
type ItemRepository interface {
	ListAvailable(ctx context.Context) ([]Item, error)
}
