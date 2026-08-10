package transporthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	stdhttp "net/http"
	"strconv"

	"github.com/L-ka080/DDB-6-case/internal/barter"
	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
)

type BarterService interface {
	SelectTransferMethod(ctx context.Context, input barter.SelectTransferMethodInput) (barter.OperationResult, error)
	MarkGiven(ctx context.Context, input barter.OperationInput) (barter.OperationResult, error)
	ConfirmReceived(ctx context.Context, input barter.OperationInput) (barter.OperationResult, error)
	Reject(ctx context.Context, input barter.RejectInput) (barter.OperationResult, error)
	MarkReturnStarted(ctx context.Context, input barter.OperationInput) (barter.OperationResult, error)
	ConfirmReturned(ctx context.Context, input barter.OperationInput) (barter.OperationResult, error)
}

type BarterHandler struct {
	service BarterService
}

func NewBarterHandler(service BarterService) *BarterHandler {
	return &BarterHandler{service: service}
}

type selectTransferMethodRequest struct {
	Method exchange.TransferMethod `json:"method"`
}

type rejectBarterRequest struct {
	Reason string `json:"reason"`
}

type barterOperationResponse struct {
	ChainID        int64                    `json:"chain_id"`
	BarterID       int64                    `json:"barter_id"`
	BarterStatus   exchange.BarterStatus    `json:"barter_status"`
	ChainStatus    exchange.Status          `json:"chain_status"`
	TransferMethod *exchange.TransferMethod `json:"transfer_method,omitempty"`
}

func (h *BarterHandler) SelectTransferMethod(c *gin.Context) {
	userID, chainID, barterID, ok := barterOperationParams(c)
	if !ok {
		return
	}

	var request selectTransferMethodRequest
	if !decodeJSONBody(c, &request) {
		return
	}

	result, err := h.service.SelectTransferMethod(c.Request.Context(), barter.SelectTransferMethodInput{
		ChainID:  chainID,
		BarterID: barterID,
		UserID:   userID,
		Method:   request.Method,
	})
	if err != nil {
		handleBarterError(c, "select barter transfer method", err)
		return
	}

	c.JSON(stdhttp.StatusOK, newBarterOperationResponse(result))
}

func (h *BarterHandler) MarkGiven(c *gin.Context) {
	h.handleOperation(c, "mark barter given", h.service.MarkGiven)
}

func (h *BarterHandler) ConfirmReceived(c *gin.Context) {
	h.handleOperation(c, "confirm barter received", h.service.ConfirmReceived)
}

func (h *BarterHandler) Reject(c *gin.Context) {
	userID, chainID, barterID, ok := barterOperationParams(c)
	if !ok {
		return
	}

	var request rejectBarterRequest
	if !decodeJSONBody(c, &request) {
		return
	}

	result, err := h.service.Reject(c.Request.Context(), barter.RejectInput{
		ChainID:  chainID,
		BarterID: barterID,
		UserID:   userID,
		Reason:   request.Reason,
	})
	if err != nil {
		handleBarterError(c, "reject barter", err)
		return
	}

	c.JSON(stdhttp.StatusOK, newBarterOperationResponse(result))
}

func (h *BarterHandler) MarkReturnStarted(c *gin.Context) {
	h.handleOperation(c, "mark barter return started", h.service.MarkReturnStarted)
}

func (h *BarterHandler) ConfirmReturned(c *gin.Context) {
	h.handleOperation(c, "confirm barter returned", h.service.ConfirmReturned)
}

func (h *BarterHandler) handleOperation(
	c *gin.Context,
	operation string,
	run func(context.Context, barter.OperationInput) (barter.OperationResult, error),
) {
	userID, chainID, barterID, ok := barterOperationParams(c)
	if !ok {
		return
	}

	result, err := run(c.Request.Context(), barter.OperationInput{
		ChainID:  chainID,
		BarterID: barterID,
		UserID:   userID,
	})
	if err != nil {
		handleBarterError(c, operation, err)
		return
	}

	c.JSON(stdhttp.StatusOK, newBarterOperationResponse(result))
}

func barterOperationParams(c *gin.Context) (int64, int64, int64, bool) {
	userID, ok := transportAuthenticatedUserID(c)
	if !ok {
		httpserver.WriteError(c, stdhttp.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return 0, 0, 0, false
	}

	chainID, ok := positivePathInt(c, "chainId")
	if !ok {
		httpserver.WriteError(c, stdhttp.StatusBadRequest, "validation_error", "chainId must be a positive integer")
		return 0, 0, 0, false
	}

	barterID, ok := positivePathInt(c, "barterId")
	if !ok {
		httpserver.WriteError(c, stdhttp.StatusBadRequest, "validation_error", "barterId must be a positive integer")
		return 0, 0, 0, false
	}

	return userID, chainID, barterID, true
}

func transportAuthenticatedUserID(c *gin.Context) (int64, bool) {
	value, ok := c.Get(middleware.UserIDContextKey)
	userID, valid := value.(int64)
	return userID, ok && valid && userID > 0
}

func positivePathInt(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	return value, err == nil && value > 0
}

func decodeJSONBody(c *gin.Context, destination any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		httpserver.WriteError(c, stdhttp.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpserver.WriteError(c, stdhttp.StatusBadRequest, "validation_error", "request body must contain one JSON object")
		return false
	}

	return true
}

func newBarterOperationResponse(result barter.OperationResult) barterOperationResponse {
	return barterOperationResponse{
		ChainID:        result.ChainID,
		BarterID:       result.BarterID,
		BarterStatus:   result.BarterStatus,
		ChainStatus:    result.ChainStatus,
		TransferMethod: result.TransferMethod,
	}
}

func handleBarterError(c *gin.Context, operation string, err error) {
	var validationError *barter.ValidationError
	switch {
	case errors.As(err, &validationError):
		httpserver.WriteError(c, stdhttp.StatusBadRequest, "validation_error", validationError.Error())
	case errors.Is(err, barter.ErrInvalidMethod):
		httpserver.WriteError(c, stdhttp.StatusBadRequest, "invalid_transfer_method", "transfer method must be meetup or delivery")
	case errors.Is(err, barter.ErrNotFound):
		httpserver.WriteError(c, stdhttp.StatusNotFound, "barter_not_found", "barter not found")
	case errors.Is(err, barter.ErrNotParticipant):
		httpserver.WriteError(c, stdhttp.StatusForbidden, "not_barter_participant", "user is not a participant of this barter")
	case errors.Is(err, barter.ErrChainStatusConflict):
		httpserver.WriteError(c, stdhttp.StatusConflict, "chain_status_conflict", "chain status does not allow this barter operation")
	case errors.Is(err, barter.ErrBarterStatusConflict):
		httpserver.WriteError(c, stdhttp.StatusConflict, "barter_status_conflict", "barter status does not allow this operation")
	case errors.Is(err, barter.ErrChainItemsConflict):
		httpserver.WriteError(c, stdhttp.StatusConflict, "chain_items_conflict", "chain items are in unexpected status")
	default:
		log.Printf("%s failed: %v", operation, err)
		httpserver.WriteError(c, stdhttp.StatusInternalServerError, "internal_error", "internal server error")
	}
}
