package httpserver

import "github.com/gin-gonic/gin"

// HealthResponse is returned by the liveness endpoint.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// ReadinessResponse is returned when all required dependencies are available.
type ReadinessResponse struct {
	Status string `json:"status" example:"ready"`
}

// ErrorDetail describes an API error.
type ErrorDetail struct {
	Code    string `json:"code" example:"validation_error"`
	Message string `json:"message" example:"request body must be valid JSON"`
}

// ErrorResponse is the common API error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// WriteError writes the common API error envelope and aborts the Gin context.
func WriteError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
