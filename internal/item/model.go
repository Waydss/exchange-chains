// Package item contains item creation domain types and use cases.
package item

import "time"

// Condition describes the physical state of an item.
type Condition string

// Supported item conditions.
const (
	ConditionNew     Condition = "new"
	ConditionLikeNew Condition = "like_new"
	ConditionGood    Condition = "good"
	ConditionFair    Condition = "fair"
	ConditionPoor    Condition = "poor"
)

// Status describes the current exchange state of an item.
type Status string

// Supported item statuses.
const (
	StatusSearching Status = "searching"
	StatusInChain   Status = "in_chain"
	StatusReserved  Status = "reserved"
	StatusInTransit Status = "in_transit"
	StatusExchanged Status = "exchanged"
	StatusArchived  Status = "archived"
)

// Item is a persisted object offered for exchange.
type Item struct {
	ID               int64
	UserID           int64
	Title            string
	Description      string
	Condition        Condition
	CategoryID       int64
	PriceMin         int64
	PriceMax         int64
	WantedCategoryID int64
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Photo is persisted metadata for an item image.
type Photo struct {
	ID        int64
	ItemID    int64
	ObjectKey string
	Position  int
	CreatedAt time.Time
}
