package item

import "context"

// ItemService describes item use cases exposed to the transport layer.
type ItemService interface {
	ListAvailable(ctx context.Context) ([]Item, error)
}
