// Package wish contains domain types for wanted items.
package wish

import "time"

// WishedItem describes what a user wants to receive in an exchange.
type WishedItem struct {
	ID            int64
	UserID        int64
	CategoryID    int64
	PriceMin      int64
	PriceMax      int64
	TitleKeywords string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
