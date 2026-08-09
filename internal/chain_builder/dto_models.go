package cb

import (
	"time"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

const rejectionRecordedStatus = "rejection_recorded"

// DeclineChainRequest contains the item that caused the chain rejection.
type DeclineChainRequest struct {
	ReceivedItemID int64 `json:"received_item_id" example:"205"`
}

// DeclineChainResponse confirms that an item rejection was recorded.
type DeclineChainResponse struct {
	ChainID        int64  `json:"chain_id" example:"77"`
	ReceivedItemID int64  `json:"received_item_id" example:"205"`
	Status         string `json:"status" example:"rejection_recorded"`
}

// ListChainsResponse is returned by the read-only chains endpoint.
type ListChainsResponse struct {
	Chains []ChainReadDTO `json:"chains"`
}

// ChainReadDTO is the read-only representation of a chain with its barters.
type ChainReadDTO struct {
	ID        int64           `json:"id"`
	RootID    int64           `json:"root_id"`
	RootWish  WishReadDTO     `json:"root_wish"`
	Signature string          `json:"signature"`
	Status    string          `json:"status"`
	Score     float64         `json:"score"`
	CreatedAt time.Time       `json:"created_at"`
	Barters   []BarterReadDTO `json:"barters"`
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

	return ChainReadDTO{
		ID:        chain.ID,
		RootID:    chain.RootID,
		RootWish:  newWishReadDTO(chain.RootWish),
		Signature: chain.Signature,
		Status:    string(chain.Status),
		Score:     chain.Score,
		CreatedAt: chain.CreatedAt,
		Barters:   barters,
	}
}
