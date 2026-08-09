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

	GetUserRatings(ctx context.Context, userId int64) ([]UserRating, error)

	ValidateBarterParticipants(ctx context.Context, barterId int64, fromUserId int64, toUserId int64) error

	UpdatePublicRating(ctx context.Context, fromid int64, toId int64, rating float64, barterId int64) (average float64, total int, err error)
	UpdateShadowRating(ctx context.Context, userId int64, reasonString string) error
}
