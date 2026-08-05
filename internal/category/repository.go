package category

import "context"

// Repository defines category lookups required by item creation.
type Repository interface {
	GetByName(ctx context.Context, name string) (Category, error)
}
