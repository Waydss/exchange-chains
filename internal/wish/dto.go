package wish

import "time"

// CategoryResponse is the public category nested in a wanted item.
type CategoryResponse struct {
	ID   int64  `json:"id" example:"3"`
	Name string `json:"name" example:"Палатки"`
}

// ReadResponse is the public representation of a wanted item.
type ReadResponse struct {
	ID            int64            `json:"id" example:"205"`
	UserID        int64            `json:"user_id" example:"6"`
	Category      CategoryResponse `json:"category"`
	PriceMin      int64            `json:"price_min" example:"40000"`
	PriceMax      int64            `json:"price_max" example:"50000"`
	TitleKeywords string           `json:"title_keywords" example:"Палатка туристическая"`
	CreatedAt     time.Time        `json:"created_at" example:"2026-08-05T15:00:00Z"`
	UpdatedAt     time.Time        `json:"updated_at" example:"2026-08-05T15:00:00Z"`
}

// ListResponse contains every wanted item owned by the authenticated user.
type ListResponse struct {
	Wishes []ReadResponse `json:"wishes"`
}
