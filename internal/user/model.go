// Package user contains the user domain model.
package user

import (
	"fmt"
	"time"
)

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

type UserRating struct {
	FromId       int64
	ToId         int64
	PublicRating float64
	BarterId     int64
}

type ShadowUpdateReason string

// FIXME We need more reasons
const (
	ProvidedExternalLink   ShadowUpdateReason = "provided external link"
	LateForItemAssesment   ShadowUpdateReason = "late for item assesment"
	InappropriateBehaviour ShadowUpdateReason = "inappropriate behaviour"
)

func getShadowScoring() map[ShadowUpdateReason]int {
	return map[ShadowUpdateReason]int{
		ProvidedExternalLink:   2,
		LateForItemAssesment:   3,
		InappropriateBehaviour: 1,
	}
}

func getShadowScore(reasonString string) (int, error) {
	scoring := getShadowScoring()

	switch reasonString {
	case "provided external link":
		return scoring[ProvidedExternalLink], nil

	case "late for item assesment":
		return scoring[LateForItemAssesment], nil

	case "inappropriate behaviour":
		return scoring[InappropriateBehaviour], nil

	default:
		return 0, fmt.Errorf("This reason does not exists")
	}
}
