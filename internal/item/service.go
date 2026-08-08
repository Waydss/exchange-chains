package item

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/L-ka080/DDB-6-case/internal/category"
)

const (
	minTitleLength       = 3
	maxTitleLength       = 200
	maxDescriptionLength = 3000
)

// Service creates items with category lookup and transactional photo metadata.
type Service struct {
	repository         Repository
	categoryRepository CategoryRepository
	fileStorage        FileStorage
	maxPhotos          int
	chainEnqueer       ChainEnqueer
}

// NewService constructs the item creation service.
func NewService(
	repository Repository,
	categoryRepository CategoryRepository,
	fileStorage FileStorage,
	maxPhotos int,
	chainEnqueer ChainEnqueer,
) *Service {
	return &Service{
		repository:         repository,
		categoryRepository: categoryRepository,
		fileStorage:        fileStorage,
		maxPhotos:          maxPhotos,
		chainEnqueer:       chainEnqueer,
	}
}

type ChainEnqueer interface {
	EnqueueChainRequest(userID, wishedItemID int64)
}

// Create validates and atomically persists an item and its photo metadata.
func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResponse, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Condition = strings.TrimSpace(input.Condition)
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	input.WantedCategoryName = strings.TrimSpace(input.WantedCategoryName)

	if err := s.validate(input); err != nil {
		return CreateResponse{}, err
	}

	itemCategory, err := s.categoryRepository.GetByName(ctx, input.CategoryName)
	if errors.Is(err, category.ErrNotFound) {
		return CreateResponse{}, ErrCategoryNotFound
	}
	if err != nil {
		return CreateResponse{}, fmt.Errorf("find item category: %w", err)
	}

	wantedCategory, err := s.categoryRepository.GetByName(ctx, input.WantedCategoryName)
	if errors.Is(err, category.ErrNotFound) {
		return CreateResponse{}, ErrWantedCategoryNotFound
	}
	if err != nil {
		return CreateResponse{}, fmt.Errorf("find wanted category: %w", err)
	}

	var createdItem Item
	var createdPhotos []Photo
	err = s.repository.WithinTransaction(ctx, func(transaction Transaction) error {
		createdItem, err = transaction.CreateItem(ctx, CreateItemParams{
			UserID:           input.UserID,
			Title:            input.Title,
			Description:      input.Description,
			Condition:        Condition(input.Condition),
			CategoryID:       itemCategory.ID,
			PriceMin:         input.PriceMin,
			PriceMax:         input.PriceMax,
			WantedCategoryID: wantedCategory.ID,
		})
		if err != nil {
			return fmt.Errorf("create item: %w", err)
		}

		storedPhotos, storageErr := s.fileStorage.SaveItemPhotos(ctx, createdItem.ID, input.Photos)
		if storageErr != nil {
			return fmt.Errorf("save item photos: %w", storageErr)
		}

		photoParams := make([]CreatePhotoParams, len(storedPhotos))
		for index, storedPhoto := range storedPhotos {
			photoParams[index] = CreatePhotoParams{URL: storedPhoto.URL, Position: index}
		}

		createdPhotos, err = transaction.CreatePhotos(ctx, createdItem.ID, photoParams)
		if err != nil {
			return fmt.Errorf("create item photo metadata: %w", err)
		}

		return nil
	})
	if err != nil {
		if createdItem.ID > 0 {
			if cleanupErr := s.fileStorage.DeleteItemPhotos(ctx, createdItem.ID); cleanupErr != nil {
				return CreateResponse{}, fmt.Errorf("%w; clean up item photos: %v", err, cleanupErr)
			}
		}

		return CreateResponse{}, err
	}

	s.chainEnqueer.EnqueueChainRequest(createdItem.UserID, createdItem.ID)

	return toCreateResponse(createdItem, itemCategory, wantedCategory, createdPhotos), nil
}

func (s *Service) validate(input CreateInput) error {
	if input.UserID <= 0 {
		return &ValidationError{Message: "authenticated user is required"}
	}

	titleLength := utf8.RuneCountInString(input.Title)
	if titleLength < minTitleLength || titleLength > maxTitleLength {
		return &ValidationError{Message: "title must contain between 3 and 200 characters"}
	}
	if utf8.RuneCountInString(input.Description) > maxDescriptionLength {
		return &ValidationError{Message: "description must not exceed 3000 characters"}
	}
	if !validCondition(Condition(input.Condition)) {
		return &ValidationError{Message: "condition must be one of: new, like_new, good, fair, poor"}
	}
	if input.CategoryName == "" {
		return &ValidationError{Message: "category_name is required"}
	}
	if input.WantedCategoryName == "" {
		return &ValidationError{Message: "wanted_category_name is required"}
	}
	if input.PriceMin <= 0 || input.PriceMax <= 0 {
		return &ValidationError{Message: "price_min and price_max must be greater than zero"}
	}
	if input.PriceMin > input.PriceMax {
		return &ValidationError{Message: "price_min must not exceed price_max"}
	}
	if len(input.Photos) > s.maxPhotos {
		return ErrTooManyPhotos
	}

	return nil
}

func validCondition(condition Condition) bool {
	switch condition {
	case ConditionNew, ConditionLikeNew, ConditionGood, ConditionFair, ConditionPoor:
		return true
	default:
		return false
	}
}

func toCreateResponse(
	created Item,
	itemCategory category.Category,
	wantedCategory category.Category,
	photos []Photo,
) CreateResponse {
	photoResponses := make([]PhotoResponse, len(photos))
	for index, photo := range photos {
		photoResponses[index] = PhotoResponse{ID: photo.ID, URL: photo.URL}
	}

	return CreateResponse{
		ID:          created.ID,
		Title:       created.Title,
		Description: created.Description,
		Condition:   created.Condition,
		Category: CategoryResponse{
			ID:   itemCategory.ID,
			Name: itemCategory.Name,
		},
		PriceMin: created.PriceMin,
		PriceMax: created.PriceMax,
		WantedCategory: CategoryResponse{
			ID:   wantedCategory.ID,
			Name: wantedCategory.Name,
		},
		Photos:    photoResponses,
		Status:    created.Status,
		CreatedAt: created.CreatedAt,
	}
}
