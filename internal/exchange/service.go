package exchange

import "context"

// ExchangeService describes exchange use cases exposed to the transport layer.
type ExchangeService interface {
	GetByID(ctx context.Context, id int64) (Exchange, error)
}
