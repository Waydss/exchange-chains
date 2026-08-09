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
	ObjectKey string
}

// CreateInput contains normalized-independent values required to create an item.
type CreateInput struct {
	UserID             int64
	Title              string
	Description        string
	Condition          string
	CategoryName       string
	PriceMin           int64
	PriceMax           int64
	WantedCategoryName string
	Photos             []PhotoUpload
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
	Photos         []PhotoResponse  `json:"photos"`
	Status         Status           `json:"status" example:"searching"`
	CreatedAt      time.Time        `json:"created_at" example:"2026-08-05T15:00:00Z"`
}
