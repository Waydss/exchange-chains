package exchange

import "context"

// ExchangeRepository describes minimal persistence access for exchange proposals.
type ExchangeRepository interface {
	GetByID(ctx context.Context, id int64) (Exchange, error)
}
