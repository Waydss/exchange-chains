package exchange

type Handler struct {
	service ExchangeService
}

// NewHandler creates an exchange HTTP handler with its dependency.
func NewHandler(service ExchangeService) *Handler {
	return &Handler{service: service}
}
