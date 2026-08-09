package cb

import (
	"errors"
	"fmt"
)

var (
	ErrChainNotFound         = errors.New("chain not found")
	ErrNotChainParticipant   = errors.New("user is not a chain participant")
	ErrChainStatusConflict   = errors.New("chain status does not allow this operation")
	ErrChainItemsUnavailable = errors.New("one or more chain items are unavailable")
)

// ValidationError describes invalid chain operation input.
type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s must be positive", e.Field)
}

func validatePositive(field string, value int64) error {
	if value <= 0 {
		return &ValidationError{Field: field}
	}
	return nil
}
