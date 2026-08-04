// Package item contains item domain types and contracts.
package item

import "time"

// Status describes the current availability of an item.
type Status string

// Supported item statuses.
const (
	StatusAvailable Status = "available"
	StatusReserved  Status = "reserved"
	StatusExchanged Status = "exchanged"
	StatusArchived  Status = "archived"
)

// Item is an object offered by its owner for exchange.
type Item struct {
	ID              int64
	OwnerID         int64
	Title           string
	Description     string
	Category        string
	DesiredCategory string
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
