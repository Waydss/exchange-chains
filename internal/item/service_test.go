package item_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/category"
	"github.com/L-ka080/DDB-6-case/internal/item"
)

var errRepository = errors.New("repository unavailable")

type fakeRepository struct {
	transactionError error
	createError      error
	photoError       error
	createParams     item.CreateItemParams
	photoParams      []item.CreatePhotoParams
	committed        bool
}

func (r *fakeRepository) WithinTransaction(_ context.Context, fn func(item.Transaction) error) error {
	if r.transactionError != nil {
		return r.transactionError
	}

	err := fn(&fakeTransaction{repository: r})
	r.committed = err == nil
	return err
}

type fakeTransaction struct {
	repository *fakeRepository
}

func (t *fakeTransaction) CreateItem(_ context.Context, params item.CreateItemParams) (item.Item, error) {
	if t.repository.createError != nil {
		return item.Item{}, t.repository.createError
	}
	t.repository.createParams = params

	return item.Item{
		ID:               101,
		UserID:           params.UserID,
		Title:            params.Title,
		Description:      params.Description,
		Condition:        params.Condition,
		CategoryID:       params.CategoryID,
		PriceMin:         params.PriceMin,
		PriceMax:         params.PriceMax,
		WantedCategoryID: params.WantedCategoryID,
		Status:           item.StatusSearching,
		CreatedAt:        time.Date(2026, time.August, 5, 15, 0, 0, 0, time.UTC),
	}, nil
}

func (t *fakeTransaction) CreatePhotos(
	_ context.Context,
	itemID int64,
	photos []item.CreatePhotoParams,
) ([]item.Photo, error) {
	if t.repository.photoError != nil {
		return nil, t.repository.photoError
	}
	t.repository.photoParams = photos

	created := make([]item.Photo, len(photos))
	for index, photo := range photos {
		created[index] = item.Photo{
			ID:       int64(501 + index),
			ItemID:   itemID,
			URL:      photo.URL,
			Position: photo.Position,
		}
	}

	return created, nil
}

type fakeCategoryRepository struct {
	categories map[string]category.Category
	errorFor   map[string]error
}

func newFakeCategoryRepository() *fakeCategoryRepository {
	return &fakeCategoryRepository{
		categories: map[string]category.Category{
			"гитары":  {ID: 12, Name: "Гитары"},
			"палатки": {ID: 25, Name: "Палатки"},
		},
		errorFor: make(map[string]error),
	}
}

func (r *fakeCategoryRepository) GetByName(_ context.Context, name string) (category.Category, error) {
	key := strings.ToLower(name)
	if err := r.errorFor[key]; err != nil {
		return category.Category{}, err
	}
	found, ok := r.categories[key]
	if !ok {
		return category.Category{}, category.ErrNotFound
	}

	return found, nil
}

type fakeStorage struct {
	error       error
	savedItemID int64
	deleteID    int64
}

func (s *fakeStorage) SaveItemPhotos(
	_ context.Context,
	itemID int64,
	photos []item.PhotoUpload,
) ([]item.StoredPhoto, error) {
	s.savedItemID = itemID
	if s.error != nil {
		return nil, s.error
	}

	stored := make([]item.StoredPhoto, len(photos))
	for index := range photos {
		stored[index] = item.StoredPhoto{URL: "/uploads/items/101/photo-" + string(rune('1'+index)) + ".jpg"}
	}

	return stored, nil
}

func (s *fakeStorage) DeleteItemPhotos(_ context.Context, itemID int64) error {
	s.deleteID = itemID
	return nil
}

type fakeChainEnqueer struct{}

func (fce fakeChainEnqueer) EnqueueChainRequest(userId, wishedItemID int64) {}

func TestCreateWithoutPhotos(t *testing.T) {
	repository := &fakeRepository{}
	storage := &fakeStorage{}
	service := item.NewService(repository, newFakeCategoryRepository(), storage, 5, fakeChainEnqueer{})

	result, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !repository.committed {
		t.Fatal("transaction was not committed")
	}
	if repository.createParams.UserID != 42 {
		t.Fatalf("persisted user ID = %d, want 42", repository.createParams.UserID)
	}
	if result.Status != item.StatusSearching || result.Category.Name != "Гитары" {
		t.Fatalf("created response = %+v", result)
	}
	if result.Photos == nil || len(result.Photos) != 0 {
		t.Fatalf("photos = %#v, want an empty JSON-compatible slice", result.Photos)
	}
}

func TestCreateWithMultiplePhotos(t *testing.T) {
	repository := &fakeRepository{}
	storage := &fakeStorage{}
	service := item.NewService(repository, newFakeCategoryRepository(), storage, 5, fakeChainEnqueer{})
	input := validCreateInput()
	input.Photos = []item.PhotoUpload{
		{Reader: bytes.NewReader([]byte("photo-1")), Size: 7},
		{Reader: bytes.NewReader([]byte("photo-2")), Size: 7},
	}

	result, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if storage.savedItemID != 101 || len(result.Photos) != 2 {
		t.Fatalf("stored item ID = %d, photos = %#v", storage.savedItemID, result.Photos)
	}
	if repository.photoParams[0].Position != 0 || repository.photoParams[1].Position != 1 {
		t.Fatalf("photo positions = %#v", repository.photoParams)
	}
}

func TestCreateRejectsUnknownCategories(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*item.CreateInput)
		wantError error
	}{
		{
			name: "unknown category",
			configure: func(input *item.CreateInput) {
				input.CategoryName = "Неизвестная"
			},
			wantError: item.ErrCategoryNotFound,
		},
		{
			name: "unknown wanted category",
			configure: func(input *item.CreateInput) {
				input.WantedCategoryName = "Неизвестная"
			},
			wantError: item.ErrWantedCategoryNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := item.NewService(&fakeRepository{}, newFakeCategoryRepository(), &fakeStorage{}, 5, fakeChainEnqueer{})
			input := validCreateInput()
			tt.configure(&input)

			_, err := service.Create(context.Background(), input)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestCreateRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*item.CreateInput)
		wantError error
	}{
		{
			name: "minimum price exceeds maximum",
			configure: func(input *item.CreateInput) {
				input.PriceMin = 60000
				input.PriceMax = 50000
			},
		},
		{
			name: "invalid condition",
			configure: func(input *item.CreateInput) {
				input.Condition = "broken"
			},
		},
		{
			name: "too many photos",
			configure: func(input *item.CreateInput) {
				input.Photos = make([]item.PhotoUpload, 6)
			},
			wantError: item.ErrTooManyPhotos,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := item.NewService(&fakeRepository{}, newFakeCategoryRepository(), &fakeStorage{}, 5, fakeChainEnqueer{})
			input := validCreateInput()
			tt.configure(&input)

			_, err := service.Create(context.Background(), input)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantError)
				}
				return
			}
			var validationError *item.ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("Create() error = %v, want ValidationError", err)
			}
		})
	}
}

func TestCreateReturnsStorageErrorsAndCleansFiles(t *testing.T) {
	tests := []error{item.ErrUnsupportedImageType, item.ErrFileStorage}
	for _, storageError := range tests {
		t.Run(storageError.Error(), func(t *testing.T) {
			repository := &fakeRepository{}
			storage := &fakeStorage{error: storageError}
			service := item.NewService(repository, newFakeCategoryRepository(), storage, 5, fakeChainEnqueer{})

			_, err := service.Create(context.Background(), validCreateInput())
			if !errors.Is(err, storageError) {
				t.Fatalf("Create() error = %v, want %v", err, storageError)
			}
			if repository.committed || storage.deleteID != 101 {
				t.Fatalf("committed = %v, cleanup item ID = %d", repository.committed, storage.deleteID)
			}
		})
	}
}

func TestCreateReturnsRepositoryError(t *testing.T) {
	repository := &fakeRepository{transactionError: errRepository}
	service := item.NewService(repository, newFakeCategoryRepository(), &fakeStorage{}, 5, fakeChainEnqueer{})

	_, err := service.Create(context.Background(), validCreateInput())
	if !errors.Is(err, errRepository) {
		t.Fatalf("Create() error = %v, want repository error", err)
	}
}

func TestCreateCleansFilesWhenPhotoMetadataFails(t *testing.T) {
	repository := &fakeRepository{photoError: errRepository}
	storage := &fakeStorage{}
	service := item.NewService(repository, newFakeCategoryRepository(), storage, 5, fakeChainEnqueer{})
	input := validCreateInput()
	input.Photos = []item.PhotoUpload{{Reader: bytes.NewReader([]byte("photo")), Size: 5}}

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, errRepository) {
		t.Fatalf("Create() error = %v, want photo repository error", err)
	}
	if repository.committed || storage.deleteID != 101 {
		t.Fatalf("committed = %v, cleanup item ID = %d", repository.committed, storage.deleteID)
	}
}

func validCreateInput() item.CreateInput {
	return item.CreateInput{
		UserID:             42,
		Title:              "  Гитара Yamaha F310, 6 струн  ",
		Description:        "  Исправна  ",
		Condition:          "good",
		CategoryName:       " Гитары ",
		PriceMin:           40000,
		PriceMax:           50000,
		WantedCategoryName: " Палатки ",
	}
}
