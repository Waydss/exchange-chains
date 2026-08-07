// Package exchange contains exchange proposal domain types and contracts.
package exchange

import (
	"time"

	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/L-ka080/DDB-6-case/internal/wish"
)

// Status describes the lifecycle state of an exchange proposal.
type Status string

// Supported exchange statuses.
const (
	StatusMatched    Status = "matched"
	StatusPending    Status = "pending"
	StatusLocked     Status = "locked"
	StatusInDelivery Status = "in_delivery"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
	StatusDisputed   Status = "disputed"
)

// ParticipantStatus describes a participant's response to a chain.
type ParticipantStatus string

// Supported participant statuses.
const (
	ParticipantPending  ParticipantStatus = "pending"
	ParticipantAccepted ParticipantStatus = "accepted"
	ParticipantDeclined ParticipantStatus = "declined"
	ParticipantExpired  ParticipantStatus = "expired"
)

// BarterStatus describes one transfer inside a chain.
type BarterStatus string

// Supported barter statuses.
const (
	BarterPending        BarterStatus = "pending"
	BarterMethodSelected BarterStatus = "method_selected"
	BarterGiven          BarterStatus = "given"
	BarterReceived       BarterStatus = "received"
	BarterRejected       BarterStatus = "rejected"
	BarterReturned       BarterStatus = "returned"
)

// TransferMethod describes how users agreed to pass an item.
type TransferMethod string

// Supported transfer methods.
const (
	TransferMethodMeetup   TransferMethod = "meetup"
	TransferMethodDelivery TransferMethod = "delivery"
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
	Decision   ParticipantStatus
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

// GivenItem is an item offered by a chain participant.
type GivenItem = item.Item

// WishedItem is a desired item declared by a chain participant.
type WishedItem = wish.WishedItem

// Chain is an ordered set of barters proposed by the matching algorithm.
type Chain struct {
	ID                int64
	RootID            int64
	RootWish          WishedItem
	Status            Status
	Barters           []Barter
	Participants      []ChainParticipant
	Signature         string
	Score             float64
	ExpiresAt         *time.Time
	LockedAt          *time.Time
	CompletedAt       *time.Time
	CancelledAt       *time.Time
	CancelledByUserID *int64
	CancelReason      string
	CreatedAt         time.Time
}

// ChainParticipant stores one user's confirmation state in a chain.
type ChainParticipant struct {
	ChainID    int64
	UserID     int64
	Status     ParticipantStatus
	AcceptedAt *time.Time
	DeclinedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Barter is one transfer inside a chain.
type Barter struct {
	ID              int64
	From            int64
	To              int64
	Give            GivenItem
	Wish            WishedItem
	Status          BarterStatus
	TransferMethod  *TransferMethod
	GivenAt         *time.Time
	ReceivedAt      *time.Time
	RejectedAt      *time.Time
	ReturnedAt      *time.Time
	RejectionReason string
	CreatedAt       time.Time
}
