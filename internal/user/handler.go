package user

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

// HandlerService contains user use cases required by the HTTP handler.
type HandlerService interface {
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
	GetByID(ctx context.Context, userID int64) (PublicUser, error)
}

// Handler exposes user authentication and profile use cases over HTTP.
type Handler struct {
	service HandlerService
}

// NewHandler creates a user HTTP handler.
func NewHandler(service HandlerService) *Handler {
	return &Handler{service: service}
}

// Register handles account registration.
func (h *Handler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return
	}

	result, err := h.service.Register(c.Request.Context(), RegisterInput(request))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Login handles credential authentication.
func (h *Handler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return
	}

	result, err := h.service.Login(c.Request.Context(), LoginInput(request))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Me returns the authenticated user's public profile.
func (h *Handler) Me(c *gin.Context) {
	userID, ok := c.Get(middleware.UserIDContextKey)
	if !ok {
		writeError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	id, ok := userID.(int64)
	if !ok || id <= 0 {
		writeError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	publicUser, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, publicUser)
}

func handleServiceError(c *gin.Context, err error) {
	var validationError *ValidationError
	switch {
	case errors.As(err, &validationError):
		writeError(c, http.StatusBadRequest, "validation_error", validationError.Message)
	case errors.Is(err, ErrInvalidCredentials):
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
	case errors.Is(err, ErrUserNotFound):
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, ErrLoginAlreadyExists):
		writeError(c, http.StatusConflict, "login_already_exists", "login is already taken")
	default:
		log.Printf("user request failed: %v", err)
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
