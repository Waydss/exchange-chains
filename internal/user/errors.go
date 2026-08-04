package user

import "errors"

var (
	// ErrUserNotFound indicates that an account does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrLoginAlreadyExists indicates a unique login conflict.
	ErrLoginAlreadyExists = errors.New("login already exists")
	// ErrInvalidCredentials intentionally covers both an unknown login and a wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// ValidationError is a client-safe input validation failure.
type ValidationError struct {
	Message string
}

// Error returns the client-safe validation message.
func (e *ValidationError) Error() string {
	return e.Message
}
