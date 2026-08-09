package cb

import (
	"time"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

// DeclineChainResponse confirms that a chain was cancelled.
type DeclineChainResponse struct {
	ChainID        int64           `json:"chain_id" example:"77"`
	ReceivedItemID int64           `json:"received_item_id" example:"205"`
	Status         exchange.Status `json:"status" example:"cancelled"`
}

// AcceptChainResponse describes the participant and chain state after acceptance.
type AcceptChainResponse struct {
	ChainID           int64                      `json:"chain_id" example:"77"`
	ParticipantStatus exchange.ParticipantStatus `json:"participant_status" example:"accepted"`
	ChainStatus       exchange.Status            `json:"chain_status" example:"pending"`
}

// ListChainsResponse is returned by the read-only chains endpoint.
type ListChainsResponse struct {
	Chains []ChainReadDTO `json:"chains"`
}

// ChainReadDTO is the read-only representation of a chain with its barters.
type ChainReadDTO struct {
	ID                int64                     `json:"id"`
	RootID            int64                     `json:"root_id"`
	RootWish          WishReadDTO               `json:"root_wish"`
	Signature         string                    `json:"signature"`
	Status            string                    `json:"status"`
	Score             float64                   `json:"score"`
	ExpiresAt         *time.Time                `json:"expires_at,omitempty"`
	LockedAt          *time.Time                `json:"locked_at,omitempty"`
	CompletedAt       *time.Time                `json:"completed_at,omitempty"`
	CancelledAt       *time.Time                `json:"cancelled_at,omitempty"`
	CancelledByUserID *int64                    `json:"cancelled_by_user_id,omitempty"`
	CancelReason      string                    `json:"cancel_reason,omitempty"`
	CreatedAt         time.Time                 `json:"created_at"`
	Participants      []ChainParticipantReadDTO `json:"participants"`
	Barters           []BarterReadDTO           `json:"barters"`
}

// ChainParticipantReadDTO exposes one participant's chain decision.
type ChainParticipantReadDTO struct {
	UserID     int64                      `json:"user_id"`
	Status     exchange.ParticipantStatus `json:"status"`
	AcceptedAt *time.Time                 `json:"accepted_at,omitempty"`
	DeclinedAt *time.Time                 `json:"declined_at,omitempty"`
	UpdatedAt  time.Time                  `json:"updated_at"`
}

// BarterReadDTO is the read-only representation of one barter inside a chain.
type BarterReadDTO struct {
	ID       int64       `json:"id"`
	Position int64       `json:"position"`
	From     int64       `json:"from_user_id"`
	To       int64       `json:"to_user_id"`
	Give     GiveReadDTO `json:"give"`
	Wish     WishReadDTO `json:"wish"`
	Status   string      `json:"status"`
}

// GiveReadDTO is the read-only representation of an offered item.
type GiveReadDTO struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Title      string `json:"title"`
	CategoryID int64  `json:"category_id"`
	PriceMin   int64  `json:"price_min"`
	PriceMax   int64  `json:"price_max"`
	Status     string `json:"status"`
}

// WishReadDTO is the read-only representation of a wanted item.
type WishReadDTO struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	CategoryID    int64  `json:"category_id"`
	PriceMin      int64  `json:"price_min"`
	PriceMax      int64  `json:"price_max"`
	TitleKeywords string `json:"title_keywords"`
}

func newWishReadDTO(wish exchange.WishedItem) WishReadDTO {
	return WishReadDTO{
		ID:            wish.ID,
		UserID:        wish.UserID,
		CategoryID:    wish.CategoryID,
		PriceMin:      wish.PriceMin,
		PriceMax:      wish.PriceMax,
		TitleKeywords: wish.TitleKeywords,
	}
}

func newChainReadDTO(chain exchange.Chain) ChainReadDTO {
	barters := make([]BarterReadDTO, 0, len(chain.Barters))
	for _, barter := range chain.Barters {
		barters = append(barters, BarterReadDTO{
			ID:       barter.ID,
			Position: int64(len(barters) + 1),
			From:     barter.From,
			To:       barter.To,
			Give: GiveReadDTO{
				ID:         barter.Give.ID,
				UserID:     barter.Give.UserID,
				Title:      barter.Give.Title,
				CategoryID: barter.Give.CategoryID,
				PriceMin:   barter.Give.PriceMin,
				PriceMax:   barter.Give.PriceMax,
				Status:     string(barter.Give.Status),
			},
			Wish:   newWishReadDTO(barter.Wish),
			Status: string(barter.Status),
		})
	}

	participants := make([]ChainParticipantReadDTO, 0, len(chain.Participants))
	for _, participant := range chain.Participants {
		participants = append(participants, ChainParticipantReadDTO{
			UserID:     participant.UserID,
			Status:     participant.Status,
			AcceptedAt: participant.AcceptedAt,
			DeclinedAt: participant.DeclinedAt,
			UpdatedAt:  participant.UpdatedAt,
		})
	}

	return ChainReadDTO{
		ID:                chain.ID,
		RootID:            chain.RootID,
		RootWish:          newWishReadDTO(chain.RootWish),
		Signature:         chain.Signature,
		Status:            string(chain.Status),
		Score:             chain.Score,
		ExpiresAt:         chain.ExpiresAt,
		LockedAt:          chain.LockedAt,
		CompletedAt:       chain.CompletedAt,
		CancelledAt:       chain.CancelledAt,
		CancelledByUserID: chain.CancelledByUserID,
		CancelReason:      chain.CancelReason,
		CreatedAt:         chain.CreatedAt,
		Participants:      participants,
		Barters:           barters,
	}
}
