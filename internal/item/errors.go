package item

import "errors"

var (
	// ErrCategoryNotFound indicates that the offered category does not exist.
	ErrCategoryNotFound = errors.New("category not found")
	// ErrWantedCategoryNotFound indicates that the requested category does not exist.
	ErrWantedCategoryNotFound = errors.New("wanted category not found")
	// ErrUnsupportedImageType indicates that an uploaded file is not JPEG, PNG, or WEBP.
	ErrUnsupportedImageType = errors.New("unsupported image type")
	// ErrImageTooLarge indicates that an uploaded image exceeds its byte limit.
	ErrImageTooLarge = errors.New("image too large")
	// ErrTooManyPhotos indicates that an item has more uploaded images than allowed.
	ErrTooManyPhotos = errors.New("too many photos")
	// ErrFileStorage indicates an infrastructure failure while persisting image files.
	ErrFileStorage = errors.New("file storage error")
)

// ValidationError is a client-safe item input validation failure.
type ValidationError struct {
	Message string
}

// Error returns the client-safe validation message.
func (e *ValidationError) Error() string {
	return e.Message
}
