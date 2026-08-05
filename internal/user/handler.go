package user

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
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

// Register godoc
//
// @Summary Регистрация пользователя
// @Description Создаёт пользователя с обязательным именем и возвращает публичный профиль с JWT-токеном доступа.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные для регистрации"
// @Success 201 {object} AuthResult
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 409 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return
	}

	result, err := h.service.Register(c.Request.Context(), RegisterInput(request))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Login godoc
//
// @Summary Вход пользователя
// @Description Проверяет логин и пароль и возвращает публичный профиль с JWT-токеном доступа.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Учётные данные"
// @Success 200 {object} AuthResult
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return
	}

	result, err := h.service.Login(c.Request.Context(), LoginInput(request))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Me godoc
//
// @Summary Профиль текущего пользователя
// @Description Возвращает публичные данные пользователя из JWT-токена.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PublicUser
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/me [get]
func (h *Handler) Me(c *gin.Context) {
	userID, ok := c.Get(middleware.UserIDContextKey)
	if !ok {
		httpserver.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	id, ok := userID.(int64)
	if !ok || id <= 0 {
		httpserver.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
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
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", validationError.Message)
	case errors.Is(err, ErrInvalidCredentials):
		httpserver.WriteError(c, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
	case errors.Is(err, ErrUserNotFound):
		httpserver.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, ErrLoginAlreadyExists):
		httpserver.WriteError(c, http.StatusConflict, "login_already_exists", "login is already taken")
	default:
		log.Printf("user request failed: %v", err)
		httpserver.WriteError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
