package user

import "context"

// CreateUserParams contains the private account data needed for persistence.
type CreateUserParams struct {
	Login        string
	PasswordHash string
	Name         string
}

// Repository defines user persistence operations required by the service.
type Repository interface {
	Create(ctx context.Context, params CreateUserParams) (User, error)
	GetByLogin(ctx context.Context, login string) (User, error)
	GetByID(ctx context.Context, id int64) (User, error)
}
