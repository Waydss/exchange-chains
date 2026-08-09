package wish

import (
	"context"
	"fmt"
)

// Service implements wanted-item reads.
type Service struct {
	repository Repository
}

// NewService creates a wanted-item service.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// ListByUserID returns all wanted items owned by the authenticated user.
func (s *Service) ListByUserID(ctx context.Context, userID int64) ([]ReadResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}

	wishes, err := s.repository.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user wishes: %w", err)
	}

	result := make([]ReadResponse, 0, len(wishes))
	for _, found := range wishes {
		result = append(result, ReadResponse{
			ID:     found.Wish.ID,
			UserID: found.Wish.UserID,
			Category: CategoryResponse{
				ID:   found.Category.ID,
				Name: found.Category.Name,
			},
			PriceMin:      found.Wish.PriceMin,
			PriceMax:      found.Wish.PriceMax,
			TitleKeywords: found.Wish.TitleKeywords,
			CreatedAt:     found.Wish.CreatedAt,
			UpdatedAt:     found.Wish.UpdatedAt,
		})
	}

	return result, nil
}
