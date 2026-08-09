package wish

import (
	"context"
	"log"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

// HandlerService contains wanted-item reads required by the HTTP handler.
type HandlerService interface {
	ListByUserID(ctx context.Context, userID int64) ([]ReadResponse, error)
}

// Handler exposes wanted-item reads over HTTP.
type Handler struct {
	service HandlerService
}

// NewHandler creates a wanted-item HTTP handler.
func NewHandler(service HandlerService) *Handler {
	return &Handler{service: service}
}

// List godoc
//
// @Summary Желания текущего пользователя
// @Description Возвращает все желаемые предметы пользователя, определённого по JWT.
// @Tags wishes
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ListResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/wishes [get]
func (h *Handler) List(c *gin.Context) {
	value, ok := c.Get(middleware.UserIDContextKey)
	userID, valid := value.(int64)
	if !ok || !valid || userID <= 0 {
		httpserver.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	wishes, err := h.service.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("list user wishes failed: %v", err)
		httpserver.WriteError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	c.JSON(http.StatusOK, ListResponse{Wishes: wishes})
}
