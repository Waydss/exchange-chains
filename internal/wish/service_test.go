package wish

import (
	"context"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/category"
)

type repositoryStub struct {
	result []Details
}

func (r repositoryStub) ListByUserID(_ context.Context, _ int64) ([]Details, error) {
	return r.result, nil
}

func TestServiceListByUserIDMapsCategory(t *testing.T) {
	service := NewService(repositoryStub{result: []Details{{
		Wish: WishedItem{
			ID:            205,
			UserID:        42,
			CategoryID:    3,
			PriceMin:      40000,
			PriceMax:      50000,
			TitleKeywords: "Палатка",
		},
		Category: category.Category{ID: 3, Name: "Палатки"},
	}}})

	result, err := service.ListByUserID(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}
	if len(result) != 1 || result[0].Category.Name != "Палатки" {
		t.Fatalf("result = %#v", result)
	}
}
