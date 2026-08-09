package item

import (
	"io"
	"time"
)

// PhotoUpload is an uploaded image stream with its declared byte size.
type PhotoUpload struct {
	Reader io.Reader
	Size   int64
}

// StoredPhoto is file metadata returned by a storage implementation.
type StoredPhoto struct {
	URL string
}

// CreateInput contains normalized-independent values required to create an item.
type CreateInput struct {
	UserID              int64
	Title               string
	Description         string
	Condition           string
	CategoryName        string
	PriceMin            int64
	PriceMax            int64
	WantedCategoryName  string
	WantedTitleKeywords string
	Photos              []PhotoUpload
}

// CategoryResponse is the public category representation nested in an item.
type CategoryResponse struct {
	ID   int64  `json:"id" example:"1"`
	Name string `json:"name" example:"Гитары"`
}

// PhotoResponse is public image metadata for an item.
type PhotoResponse struct {
	ID  int64  `json:"id" example:"501"`
	URL string `json:"url" example:"/uploads/items/101/7d9f03c7.webp"`
}

// WishResponse is the wanted item created together with an offered item.
type WishResponse struct {
	ID            int64            `json:"id" example:"205"`
	Category      CategoryResponse `json:"category"`
	PriceMin      int64            `json:"price_min" example:"40000"`
	PriceMax      int64            `json:"price_max" example:"50000"`
	TitleKeywords string           `json:"title_keywords" example:"Палатка туристическая"`
}

// CreateResponse is returned after an item has been created successfully.
type CreateResponse struct {
	ID             int64            `json:"id" example:"101"`
	Title          string           `json:"title" example:"Гитара Yamaha F310, 6 струн"`
	Description    string           `json:"description" example:"Исправна, есть небольшие царапины"`
	Condition      Condition        `json:"condition" example:"good"`
	Category       CategoryResponse `json:"category"`
	PriceMin       int64            `json:"price_min" example:"40000"`
	PriceMax       int64            `json:"price_max" example:"50000"`
	WantedCategory CategoryResponse `json:"wanted_category"`
	Wish           WishResponse     `json:"wish"`
	Photos         []PhotoResponse  `json:"photos"`
	Status         Status           `json:"status" example:"searching"`
	CreatedAt      time.Time        `json:"created_at" example:"2026-08-05T15:00:00Z"`
}

// ReadResponse is the complete public representation of a persisted item.
type ReadResponse struct {
	ID             int64            `json:"id" example:"101"`
	UserID         int64            `json:"user_id" example:"6"`
	Title          string           `json:"title" example:"Гитара Yamaha F310, 6 струн"`
	Description    string           `json:"description" example:"Исправна, есть небольшие царапины"`
	Condition      Condition        `json:"condition" example:"good"`
	Category       CategoryResponse `json:"category"`
	PriceMin       int64            `json:"price_min" example:"40000"`
	PriceMax       int64            `json:"price_max" example:"50000"`
	WantedCategory CategoryResponse `json:"wanted_category"`
	Photos         []PhotoResponse  `json:"photos"`
	Status         Status           `json:"status" example:"searching"`
	CreatedAt      time.Time        `json:"created_at" example:"2026-08-05T15:00:00Z"`
	UpdatedAt      time.Time        `json:"updated_at" example:"2026-08-05T15:00:00Z"`
}

// ListResponse contains every item owned by the authenticated user.
type ListResponse struct {
	Items []ReadResponse `json:"items"`
}
