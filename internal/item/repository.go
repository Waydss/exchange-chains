package item

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/category"
	"github.com/L-ka080/DDB-6-case/internal/wish"
)

// CreateItemParams contains database fields for a new item.
type CreateItemParams struct {
	UserID           int64
	Title            string
	Description      string
	Condition        Condition
	CategoryID       int64
	PriceMin         int64
	PriceMax         int64
	WantedCategoryID int64
}

// CreatePhotoParams contains database fields for a stored photo.
type CreatePhotoParams struct {
	ObjectKey string
	Position  int
}

// CreateWishParams contains database fields for a wanted item.
type CreateWishParams struct {
	UserID        int64
	CategoryID    int64
	PriceMin      int64
	PriceMax      int64
	TitleKeywords string
}

// Details contains an item together with the data needed by read endpoints.
type Details struct {
	Item           Item
	Category       category.Category
	WantedCategory category.Category
	Photos         []Photo
}

// Transaction contains item writes that must commit atomically.
type Transaction interface {
	CreateItem(ctx context.Context, params CreateItemParams) (Item, error)
	CreatePhotos(ctx context.Context, itemID int64, photos []CreatePhotoParams) ([]Photo, error)
	CreateWish(ctx context.Context, params CreateWishParams) (wish.WishedItem, error)
}

// Repository coordinates transactional item persistence.
type Repository interface {
	WithinTransaction(ctx context.Context, fn func(Transaction) error) error
	ListByUserID(ctx context.Context, userID int64) ([]Details, error)
	GetByID(ctx context.Context, itemID int64) (Details, error)
	UpdatePhoto(ctx context.Context, itemID, photoID int64, objectKey string) (Photo, error)
	DeletePhoto(ctx context.Context, itemID, photoID int64) error
}

// CategoryRepository is the category lookup capability required by the service.
type CategoryRepository interface {
	GetByName(ctx context.Context, name string) (category.Category, error)
}

// FileStorage persists and removes uploaded item photos.
type FileStorage interface {
	SaveItemPhotos(ctx context.Context, itemID int64, photos []PhotoUpload) ([]StoredPhoto, error)
	DeleteItemPhotos(ctx context.Context, itemID int64) error
	DeletePhotos(ctx context.Context, objectKeys []string) error
	PublicURL(objectKey string) string
}
