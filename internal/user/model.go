// Package user contains the user domain model.
package user

import "time"

// User contains persisted account data, including fields that must remain private.
type User struct {
	ID           int64
	Login        string
	PasswordHash string
	Name         string
	PublicRating float64
	ShadowRating int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
