package barter

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound             = errors.New("barter not found")
	ErrNotParticipant       = errors.New("user is not a barter participant")
	ErrChainStatusConflict  = errors.New("chain status does not allow this barter operation")
	ErrBarterStatusConflict = errors.New("barter status does not allow this operation")
	ErrChainItemsConflict   = errors.New("chain items are in unexpected status")
	ErrInvalidMethod        = errors.New("invalid transfer method")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func validatePositive(field string, value int64) error {
	if value <= 0 {
		return &ValidationError{Message: fmt.Sprintf("%s must be positive", field)}
	}
	return nil
}
