// Package category contains category domain types and persistence contracts.
package category

import "time"

// Category identifies an item category available in the catalogue.
type Category struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}
