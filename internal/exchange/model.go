// Package exchange contains exchange proposal domain types and contracts.
package exchange

import "time"

// Status describes the lifecycle state of an exchange proposal.
type Status string

// Supported exchange statuses.
const (
	StatusPending   Status = "pending"
	StatusReady     Status = "ready"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
	StatusCompleted Status = "completed"
)

// ParticipantDecision describes a participant's response to a proposal.
type ParticipantDecision string

// Supported participant decisions.
const (
	DecisionPending  ParticipantDecision = "pending"
	DecisionAccepted ParticipantDecision = "accepted"
	DecisionDeclined ParticipantDecision = "declined"
)

// Exchange is a proposed cyclic exchange.
type Exchange struct {
	ID        int64
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
}

// Participant associates a user and their decision with an exchange.
type Participant struct {
	ExchangeID int64
	UserID     int64
	Decision   ParticipantDecision
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Transfer records who gives which item to whom within an exchange.
type Transfer struct {
	ExchangeID int64
	ItemID     int64
	FromUserID int64
	ToUserID   int64
	CreatedAt  time.Time
}
