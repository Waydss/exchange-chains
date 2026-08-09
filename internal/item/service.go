package item

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/L-ka080/DDB-6-case/internal/category"
	"github.com/L-ka080/DDB-6-case/internal/wish"
)

const (
	minTitleLength               = 3
	maxTitleLength               = 200
	maxDescriptionLength         = 3000
	maxWantedTitleKeywordsLength = 200
)

// Service creates offered items and wishes with category lookup and photo metadata.
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

// Create validates and atomically persists an item, its wish, and photo metadata.
func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResponse, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Condition = strings.TrimSpace(input.Condition)
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	input.WantedCategoryName = strings.TrimSpace(input.WantedCategoryName)
	input.WantedTitleKeywords = strings.TrimSpace(input.WantedTitleKeywords)

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
	var createdWish wish.WishedItem
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
			photoParams[index] = CreatePhotoParams{ObjectKey: storedPhoto.ObjectKey, Position: index}
		}

		createdPhotos, err = transaction.CreatePhotos(ctx, createdItem.ID, photoParams)
		if err != nil {
			return fmt.Errorf("create item photo metadata: %w", err)
		}

		createdWish, err = transaction.CreateWish(ctx, CreateWishParams{
			UserID:        input.UserID,
			CategoryID:    wantedCategory.ID,
			PriceMin:      input.PriceMin,
			PriceMax:      input.PriceMax,
			TitleKeywords: input.WantedTitleKeywords,
		})
		if err != nil {
			return fmt.Errorf("create wanted item: %w", err)
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

	s.chainEnqueer.EnqueueChainRequest(createdItem.UserID, createdWish.ID)

	return s.toCreateResponse(createdItem, createdWish, itemCategory, wantedCategory, createdPhotos), nil
}

// ListByUserID returns all items owned by the authenticated user.
func (s *Service) ListByUserID(ctx context.Context, userID int64) ([]ReadResponse, error) {
	if userID <= 0 {
		return nil, &ValidationError{Message: "authenticated user is required"}
	}

	items, err := s.repository.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user items: %w", err)
	}

	return s.toReadResponses(items), nil
}

// GetByID returns an item visible to any authenticated user.
func (s *Service) GetByID(ctx context.Context, itemID int64) (ReadResponse, error) {
	if itemID <= 0 {
		return ReadResponse{}, &ValidationError{Message: "itemId must be a positive integer"}
	}

	found, err := s.repository.GetByID(ctx, itemID)
	if err != nil {
		return ReadResponse{}, fmt.Errorf("get item: %w", err)
	}

	return s.toReadResponse(found), nil
}

// AddPhotos validates ownership and appends photos to an existing item.
func (s *Service) AddPhotos(ctx context.Context, input AddPhotosInput) (AddPhotosResult, error) {
	if input.UserID <= 0 {
		return AddPhotosResult{}, &ValidationError{Message: "authenticated user is required"}
	}
	if input.ItemID <= 0 {
		return AddPhotosResult{}, &ValidationError{Message: "itemId must be a positive integer"}
	}
	if len(input.Photos) == 0 {
		return AddPhotosResult{}, &ValidationError{Message: "at least one photo is required"}
	}
	if len(input.Photos) > s.maxPhotos {
		return AddPhotosResult{}, ErrTooManyPhotos
	}

	found, err := s.repository.GetByID(ctx, input.ItemID)
	if err != nil {
		return AddPhotosResult{}, fmt.Errorf("get item: %w", err)
	}
	if found.Item.UserID != input.UserID {
		return AddPhotosResult{}, ErrForbidden
	}
	if len(found.Photos)+len(input.Photos) > s.maxPhotos {
		return AddPhotosResult{}, ErrTooManyPhotos
	}

	storedPhotos, err := s.fileStorage.SaveItemPhotos(ctx, input.ItemID, input.Photos)
	if err != nil {
		return AddPhotosResult{}, err
	}

	nextPosition := nextPhotoPosition(found.Photos)
	photoParams := make([]CreatePhotoParams, len(storedPhotos))
	storedObjectKeys := make([]string, len(storedPhotos))
	for index, storedPhoto := range storedPhotos {
		photoParams[index] = CreatePhotoParams{ObjectKey: storedPhoto.ObjectKey, Position: nextPosition + index}
		storedObjectKeys[index] = storedPhoto.ObjectKey
	}

	var createdPhotos []Photo
	err = s.repository.WithinTransaction(ctx, func(transaction Transaction) error {
		var createErr error
		createdPhotos, createErr = transaction.CreatePhotos(ctx, input.ItemID, photoParams)
		if createErr != nil {
			return fmt.Errorf("create item photo metadata: %w", createErr)
		}
		return nil
	})
	if err != nil {
		if cleanupErr := s.fileStorage.DeletePhotos(ctx, storedObjectKeys); cleanupErr != nil {
			return AddPhotosResult{}, fmt.Errorf("%w; clean up item photos: %v", err, cleanupErr)
		}
		return AddPhotosResult{}, err
	}

	return AddPhotosResult{Photos: s.toPhotoResults(createdPhotos)}, nil
}

// ReplacePhoto validates ownership and swaps one photo object while preserving its metadata ID.
func (s *Service) ReplacePhoto(ctx context.Context, input ReplacePhotoInput) (ReplacePhotoResult, error) {
	if input.UserID <= 0 {
		return ReplacePhotoResult{}, &ValidationError{Message: "authenticated user is required"}
	}
	if input.ItemID <= 0 {
		return ReplacePhotoResult{}, &ValidationError{Message: "itemId must be a positive integer"}
	}
	if input.PhotoID <= 0 {
		return ReplacePhotoResult{}, &ValidationError{Message: "photoId must be a positive integer"}
	}
	if input.Photo.Reader == nil {
		return ReplacePhotoResult{}, &ValidationError{Message: "photo is required"}
	}

	found, err := s.repository.GetByID(ctx, input.ItemID)
	if err != nil {
		return ReplacePhotoResult{}, fmt.Errorf("get item: %w", err)
	}
	if found.Item.UserID != input.UserID {
		return ReplacePhotoResult{}, ErrForbidden
	}

	var oldPhoto Photo
	for _, photo := range found.Photos {
		if photo.ID == input.PhotoID {
			oldPhoto = photo
			break
		}
	}
	if oldPhoto.ID == 0 {
		return ReplacePhotoResult{}, ErrPhotoNotFound
	}

	storedPhotos, err := s.fileStorage.SaveItemPhotos(ctx, input.ItemID, []PhotoUpload{input.Photo})
	if err != nil {
		return ReplacePhotoResult{}, err
	}
	newObjectKey := storedPhotos[0].ObjectKey

	updatedPhoto, err := s.repository.UpdatePhoto(ctx, input.ItemID, input.PhotoID, newObjectKey)
	if err != nil {
		if cleanupErr := s.fileStorage.DeletePhotos(ctx, []string{newObjectKey}); cleanupErr != nil {
			return ReplacePhotoResult{}, fmt.Errorf("%w; clean up replacement photo: %v", err, cleanupErr)
		}
		return ReplacePhotoResult{}, fmt.Errorf("replace item photo metadata: %w", err)
	}

	if err := s.fileStorage.DeletePhotos(ctx, []string{oldPhoto.ObjectKey}); err != nil {
		return ReplacePhotoResult{}, err
	}

	return ReplacePhotoResult{Photo: s.toPhotoResult(updatedPhoto)}, nil
}

// DeletePhoto validates ownership and removes a photo from an existing item.
func (s *Service) DeletePhoto(ctx context.Context, userID, itemID, photoID int64) error {
	if userID <= 0 {
		return &ValidationError{Message: "authenticated user is required"}
	}
	if itemID <= 0 {
		return &ValidationError{Message: "itemId must be a positive integer"}
	}
	if photoID <= 0 {
		return &ValidationError{Message: "photoId must be a positive integer"}
	}

	found, err := s.repository.GetByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}
	if found.Item.UserID != userID {
		return ErrForbidden
	}

	var photoToDelete Photo
	for _, photo := range found.Photos {
		if photo.ID == photoID {
			photoToDelete = photo
			break
		}
	}
	if photoToDelete.ID == 0 {
		return ErrPhotoNotFound
	}

	if err := s.repository.DeletePhoto(ctx, itemID, photoID); err != nil {
		return fmt.Errorf("delete item photo metadata: %w", err)
	}
	if err := s.fileStorage.DeletePhotos(ctx, []string{photoToDelete.ObjectKey}); err != nil {
		return err
	}

	return nil
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
	keywordsLength := utf8.RuneCountInString(input.WantedTitleKeywords)
	if keywordsLength == 0 || keywordsLength > maxWantedTitleKeywordsLength {
		return &ValidationError{Message: "wanted_title_keywords must contain between 1 and 200 characters"}
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

func (s *Service) toCreateResponse(
	created Item,
	createdWish wish.WishedItem,
	itemCategory category.Category,
	wantedCategory category.Category,
	photos []Photo,
) CreateResponse {
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
		Wish: WishResponse{
			ID: createdWish.ID,
			Category: CategoryResponse{
				ID:   wantedCategory.ID,
				Name: wantedCategory.Name,
			},
			PriceMin:      createdWish.PriceMin,
			PriceMax:      createdWish.PriceMax,
			TitleKeywords: createdWish.TitleKeywords,
		},
		Photos:    s.toPhotoResponses(photos),
		Status:    created.Status,
		CreatedAt: created.CreatedAt,
	}
}

func (s *Service) toReadResponses(items []Details) []ReadResponse {
	responses := make([]ReadResponse, 0, len(items))
	for _, found := range items {
		responses = append(responses, s.toReadResponse(found))
	}
	return responses
}

func (s *Service) toReadResponse(found Details) ReadResponse {
	return ReadResponse{
		ID:          found.Item.ID,
		UserID:      found.Item.UserID,
		Title:       found.Item.Title,
		Description: found.Item.Description,
		Condition:   found.Item.Condition,
		Category: CategoryResponse{
			ID:   found.Category.ID,
			Name: found.Category.Name,
		},
		PriceMin: found.Item.PriceMin,
		PriceMax: found.Item.PriceMax,
		WantedCategory: CategoryResponse{
			ID:   found.WantedCategory.ID,
			Name: found.WantedCategory.Name,
		},
		Photos:    s.toPhotoResponses(found.Photos),
		Status:    found.Item.Status,
		CreatedAt: found.Item.CreatedAt,
		UpdatedAt: found.Item.UpdatedAt,
	}
}

func (s *Service) toPhotoResponses(photos []Photo) []PhotoResponse {
	responses := make([]PhotoResponse, len(photos))
	for index, photo := range photos {
		responses[index] = s.toPhotoResponse(photo)
	}
	return responses
}

func (s *Service) toPhotoResponse(photo Photo) PhotoResponse {
	return PhotoResponse{ID: photo.ID, URL: s.fileStorage.PublicURL(photo.ObjectKey)}
}

func (s *Service) toPhotoResults(photos []Photo) []PhotoResult {
	results := make([]PhotoResult, len(photos))
	for index, photo := range photos {
		results[index] = s.toPhotoResult(photo)
	}
	return results
}

func (s *Service) toPhotoResult(photo Photo) PhotoResult {
	return PhotoResult{ID: photo.ID, URL: s.fileStorage.PublicURL(photo.ObjectKey)}
}

func nextPhotoPosition(photos []Photo) int {
	nextPosition := 0
	for _, photo := range photos {
		if photo.Position >= nextPosition {
			nextPosition = photo.Position + 1
		}
	}
	return nextPosition
}
