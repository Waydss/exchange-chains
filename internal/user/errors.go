package user

import "errors"

var (
	// ErrUserNotFound indicates that an account does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrLoginAlreadyExists indicates a unique login conflict.
	ErrLoginAlreadyExists = errors.New("login already exists")
	// ErrInvalidCredentials intentionally covers both an unknown login and a wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrBarterNotFound indicates that a barter does not exist.
	ErrBarterNotFound = errors.New("barter not found")
	// ErrBarterUsersMismatch indicates that the users are not both participants of the barter.
	ErrBarterUsersMismatch = errors.New("users do not belong to the barter")
	// ErrRatingAlreadyExists indicates that a rating record for the same from/to/barter already exists.
	ErrRatingAlreadyExists = errors.New("rating already exists")
)

// ValidationError is a client-safe input validation failure.
type ValidationError struct {
	Message string
}

// Error returns the client-safe validation message.
func (e *ValidationError) Error() string {
	return e.Message
}
