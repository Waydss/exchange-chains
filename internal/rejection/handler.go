package rejection

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

// HandlerService contains the chain-decline use case required by the HTTP handler.
type HandlerService interface {
	Execute(ctx context.Context, userID, chainID, receivedItemID int64) error
}

// Handler exposes item rejection over HTTP.
type Handler struct {
	service HandlerService
}

// NewHandler creates a chain-decline HTTP handler.
func NewHandler(service HandlerService) *Handler {
	return &Handler{service: service}
}

// DeclineChain godoc
//
// @Summary Отказ от цепочки обмена
// @Description Фиксирует в Redis временный отказ текущего пользователя от товара, который он должен был получить. Состояние цепочки в PostgreSQL пока не изменяется.
// @Tags chains
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param chainId path integer true "ID цепочки"
// @Param request body DeclineChainRequest true "Получаемый товар, вызвавший отказ"
// @Success 200 {object} DeclineChainResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/{chainId}/decline [post]
func (h *Handler) DeclineChain(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.UserIDContextKey)
	userID, validUserID := userIDValue.(int64)
	if !ok || !validUserID || userID <= 0 {
		httpserver.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	chainID, err := strconv.ParseInt(c.Param("chainId"), 10, 64)
	if err != nil || chainID <= 0 {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", "chainId must be a positive integer")
		return
	}

	var request DeclineChainRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return
	}
	if request.ReceivedItemID <= 0 {
		httpserver.WriteError(c, http.StatusBadRequest, "validation_error", "received_item_id must be positive")
		return
	}

	if err := h.service.Execute(c.Request.Context(), userID, chainID, request.ReceivedItemID); err != nil {
		var validationError *ValidationError
		if errors.As(err, &validationError) {
			httpserver.WriteError(c, http.StatusBadRequest, "validation_error", validationError.Error())
			return
		}
		log.Printf("decline chain request failed: %v", err)
		httpserver.WriteError(c, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	c.JSON(http.StatusOK, DeclineChainResponse{
		ChainID:        chainID,
		ReceivedItemID: request.ReceivedItemID,
		Status:         rejectionRecordedStatus,
	})
}
