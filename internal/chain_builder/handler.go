package cb

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/chain_builder/entities"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Handler exposes chain building over HTTP.
type Handler struct {
	service *Service
}

// NewHandler creates a chain-builder HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}


func (h *Handler) CreateChainsForItem(ctx *gin.Context) {
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()

	request := CreateChainRequest{}
	if err := decoder.Decode(&request); err != nil {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "request body must be valid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "request body must contain one JSON object")
		return
	}

	if request.UserId == 0 || request.WishedItemId == 0 {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id and wished_item_id must be positive")
		return
	}

	foundAccount, err := h.service.repository.GetAccount(ctx.Request.Context(), request.UserId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.WriteError(ctx, http.StatusNotFound, "account_not_found", "account not found")
			return
		}
		httpserver.WriteError(ctx, http.StatusInternalServerError, "database_error", "could not load account")
		return
	}

	wishExists := false
	for _, wish := range foundAccount.Wishes {
		if wish.ID == request.WishedItemId {
			wishExists = true
			break
		}
	}
	if !wishExists {
		httpserver.WriteError(ctx, http.StatusNotFound, "wish_not_found", "wished item not found for account")
		return
	}

	h.service.EnqueueChainRequest(ChainRequest{UserAccount: &foundAccount, WishedItemId: request.WishedItemId})

	ctx.Status(http.StatusAccepted)
}

type CreateChainRequest struct {
	UserId       uint `json:"user_id"`
	WishedItemId uint `json:"wished_item_id"`
}

type CreateChainResponse struct {
	Chains []ChainDTO `json:"chains"`
}

type ChainDTO struct {
	RootId   uint                `json:"root_id"`
	RootWish entities.WishedItem `json:"root_wish"`
	Barters  []entities.Barter   `json:"barters"`
	Complete bool                `json:"complete"`
	Score    float64             `json:"score"`
}

type ChainRequest struct {
	UserAccount  *entities.Account `json:"user_id"`
	WishedItemId uint              `json:"wished_item"`
}
