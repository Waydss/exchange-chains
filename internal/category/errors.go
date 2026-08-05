package category

import "errors"

// ErrNotFound indicates that a requested category does not exist.
var ErrNotFound = errors.New("category not found")
