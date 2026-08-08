package cb

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
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

// CreateChainsForItem godoc
//
// @Summary Запуск построения цепочек для желания
// @Description Ставит запрос на построение цепочек обмена для указанного желания в очередь. Проверяет существование аккаунта и принадлежность желания этому аккаунту, затем отвечает 202 Accepted; построение выполняется асинхронно воркерами Service, результат пишется в БД по мере нахождения цепей.
// @Tags chains
// @Accept json
// @Produce json
// @Param user_id body int64 true "Идентификатор пользователя"
// @Param wished_item_id body int64 true "Идентификатор желания"
// @Success 202
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chain [post]
// @Security BearerAuth
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

	h.service.EnqueueChainRequest(request.UserId, request.WishedItemId)

	ctx.Status(http.StatusAccepted)
}

type CreateChainRequest struct {
	UserId       int64 `json:"user_id"`
	WishedItemId int64 `json:"wished_item_id"`
}

// GetChainsByItemRequest holds the query parameters of the by-item chain endpoint.
type GetChainsByItemRequest struct {
	UserID       int64 `form:"user_id"`
	WishedItemID int64 `form:"wished_item_id"`
}

// GetChainsByUserRequest holds the query parameters of the by-user chain endpoint.
type GetChainsByUserRequest struct {
	UserID int64 `form:"user_id"`
}

type CreateChainResponse struct {
	Chains []ChainDTO `json:"chains"`
}

type ChainDTO struct {
	RootId   uint                `json:"root_id"`
	RootWish exchange.WishedItem `json:"root_wish"`
	Barters  []exchange.Barter   `json:"barters"`
	Complete bool                `json:"complete"`
	Score    float64             `json:"score"`
}

type ChainRequest struct {
	UserID       int64 `json:"user_id"`
	WishedItemId int64 `json:"wished_item"`
}

// ListChains godoc
//
// @Summary Список цепочек
// @Description Возвращает все цепочки обмена с вложенными бартерами.
// @Tags chains
// @Produce json
// @Success 200 {object} ListChainsResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains [get]
func (h *Handler) ListChains(ctx *gin.Context) {
	chains, err := h.service.repository.ListChains(ctx.Request.Context())
	if err != nil {
		httpserver.WriteError(ctx, http.StatusInternalServerError, "database_error", "could not load chains")
		return
	}

	response := ListChainsResponse{
		Chains: make([]ChainReadDTO, 0, len(chains)),
	}
	for _, chain := range chains {
		response.Chains = append(response.Chains, newChainReadDTO(chain))
	}

	ctx.JSON(http.StatusOK, response)
}

// GetChainsByItem godoc
//
// @Summary Цепочки для желания
// @Description Возвращает цепочки обмена, построенные для указанного желания пользователя. Идентификаторы передаются как query-параметры user_id и wished_item_id.
// @Tags chains
// @Produce json
// @Param user_id query int true "Идентификатор пользователя"
// @Param wished_item_id query int true "Идентификатор желания"
// @Success 200 {object} ListChainsResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/by-item [get]
func (h *Handler) GetChainsByItem(ctx *gin.Context) {
	var request GetChainsByItemRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id and wished_item_id must be positive integers")
		return
	}
	if request.UserID == 0 || request.WishedItemID == 0 {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id and wished_item_id must be positive integers")
		return
	}

	chains, err := h.service.GetChainsByItem(ctx.Request.Context(), request.UserID, request.WishedItemID)
	if err != nil {
		httpserver.WriteError(ctx, http.StatusInternalServerError, "database_error", "could not load chains")
		return
	}

	response := ListChainsResponse{
		Chains: make([]ChainReadDTO, 0, len(chains)),
	}
	for _, chain := range chains {
		response.Chains = append(response.Chains, newChainReadDTO(chain))
	}

	ctx.JSON(http.StatusOK, response)
}

// GetChainsByUser godoc
//
// @Summary Цепочки пользователя
// @Description Возвращает цепочки обмена, в которых пользователь является корневым участником или участником бартера. Идентификатор передаётся как query-параметр user_id.
// @Tags chains
// @Produce json
// @Param user_id query int true "Идентификатор пользователя"
// @Success 200 {object} ListChainsResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/by-user [get]
func (h *Handler) GetChainsByUser(ctx *gin.Context) {
	var request GetChainsByUserRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id must be a positive integer")
		return
	}
	if request.UserID == 0 {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id must be a positive integer")
		return
	}

	chains, err := h.service.GetChainsByUser(ctx.Request.Context(), request.UserID)
	if err != nil {
		httpserver.WriteError(ctx, http.StatusInternalServerError, "database_error", "could not load chains")
		return
	}

	response := ListChainsResponse{
		Chains: make([]ChainReadDTO, 0, len(chains)),
	}
	for _, chain := range chains {
		response.Chains = append(response.Chains, newChainReadDTO(chain))
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) AcceptChain(ctx *gin.Context) {
	httpserver.WriteError(ctx, http.StatusInternalServerError, "not_implemented", "not implemented")

	// h.service.AcceptChain(ctx, )
}

func (h *Handler) DiscardChain(ctx *gin.Context) {

	httpserver.WriteError(ctx, http.StatusInternalServerError, "not_implemented", "not implemented")
}
