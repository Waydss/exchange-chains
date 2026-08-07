package rejection

import "fmt"

// ValidationError describes invalid rejection or decline input.
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
