package item

// Handler will expose item use cases over HTTP in a later stage.
type Handler struct {
	service ItemService
}

// NewHandler creates an item HTTP handler with its dependency.
func NewHandler(service ItemService) *Handler {
	return &Handler{service: service}
}
