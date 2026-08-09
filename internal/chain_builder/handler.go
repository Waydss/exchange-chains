package cb

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/gin-gonic/gin"
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

	publicRating, shadowRating, err := h.service.userRequester.GetUserCandidateBase(ctx.Request.Context(), request.UserId)

	if err != nil {
		httpserver.WriteError(ctx, http.StatusNotFound, "not_found", "user account has not been found")
		return
	}

	candidate := userCandidate{
		ID:           request.UserId,
		PublicRating: publicRating,
		ShadowRating: shadowRating,
	}
	err = h.service.repository.FillRootCandidate(ctx.Request.Context(), &candidate)

	if err != nil {
		httpserver.WriteError(ctx, http.StatusInternalServerError, "database_error", "could not load account")
		return
	}

	wishExists := false
	for _, wish := range candidate.Wishes {
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

// ListChains returns every persisted chain with its nested data.
// It is kept for internal tooling and is not exposed by the HTTP router.
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
// @Summary Все цепочки конкретного желания пользователя
// @Description Возвращает все цепочки обмена, построенные для конкретного желания пользователя. Идентификаторы передаются как query-параметры user_id и wished_item_id.
// @Tags chains
// @Produce json
// @Security BearerAuth
// @Param user_id query int true "Идентификатор пользователя"
// @Param wished_item_id query int true "Идентификатор желания"
// @Success 200 {object} ListChainsResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/by-item [get]
func (h *Handler) GetChainsByItem(ctx *gin.Context) {
	var request GetChainsByItemRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id and wished_item_id must be positive integers")
		return
	}
	if request.UserID <= 0 || request.WishedItemID <= 0 {
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
// @Security BearerAuth
// @Param user_id query int true "Идентификатор пользователя"
// @Success 200 {object} ListChainsResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/by-user [get]
func (h *Handler) GetChainsByUser(ctx *gin.Context) {
	var request GetChainsByUserRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "user_id must be a positive integer")
		return
	}
	if request.UserID <= 0 {
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

// GetChainByID godoc
//
// @Summary Получение цепочки по ID
// @Description Возвращает одну цепочку со всеми участниками и обменами. Доступ разрешён только участнику этой цепочки.
// @Tags chains
// @Produce json
// @Security BearerAuth
// @Param chainId path integer true "ID цепочки"
// @Success 200 {object} ChainReadDTO
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 403 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/{chainId} [get]
func (h *Handler) GetChainByID(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		httpserver.WriteError(ctx, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	chainID, ok := chainIDFromPath(ctx)
	if !ok {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "chainId must be a positive integer")
		return
	}

	chain, err := h.service.GetChainByID(ctx.Request.Context(), chainID, userID)
	if err != nil {
		handleChainOperationError(ctx, "get chain by ID", err)
		return
	}

	ctx.JSON(http.StatusOK, newChainReadDTO(chain))
}

// AcceptChain godoc
//
// @Summary Принятие цепочки обмена
// @Description Фиксирует согласие текущего пользователя. После согласия всех участников блокирует цепочку и резервирует участвующие предметы.
// @Tags chains
// @Produce json
// @Security BearerAuth
// @Param chainId path integer true "ID цепочки"
// @Success 200 {object} AcceptChainResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 403 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Failure 409 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/{chainId}/accept [post]
func (h *Handler) AcceptChain(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		httpserver.WriteError(ctx, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	chainID, ok := chainIDFromPath(ctx)
	if !ok {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "chainId must be a positive integer")
		return
	}

	result, err := h.service.AcceptChain(ctx.Request.Context(), chainID, userID)
	if err != nil {
		handleChainOperationError(ctx, "accept chain request", err)
		return
	}

	ctx.JSON(http.StatusOK, AcceptChainResponse{
		ChainID:           result.ChainID,
		ParticipantStatus: result.ParticipantStatus,
		ChainStatus:       result.ChainStatus,
	})
}

// DeclineChain godoc
//
// @Summary Отказ от цепочки обмена
// @Description По текущему пользователю и цепочке определяет получаемый предмет, отмечает пользователя как отказавшегося, отменяет цепочку в PostgreSQL и временно сохраняет отклонённый предмет в Redis.
// @Tags chains
// @Produce json
// @Security BearerAuth
// @Param chainId path integer true "ID цепочки"
// @Success 200 {object} DeclineChainResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 403 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Failure 409 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /api/v1/chains/{chainId}/decline [post]
func (h *Handler) DeclineChain(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		httpserver.WriteError(ctx, http.StatusUnauthorized, "invalid_token", "invalid or expired access token")
		return
	}

	chainID, ok := chainIDFromPath(ctx)
	if !ok {
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", "chainId must be a positive integer")
		return
	}

	result, err := h.service.DeclineChain(ctx.Request.Context(), chainID, userID)
	if err != nil {
		handleChainOperationError(ctx, "decline chain request", err)
		return
	}

	ctx.JSON(http.StatusOK, DeclineChainResponse{
		ChainID:        result.ChainID,
		ReceivedItemID: result.ReceivedItemID,
		Status:         result.ChainStatus,
	})
}

func authenticatedUserID(ctx *gin.Context) (int64, bool) {
	value, ok := ctx.Get(middleware.UserIDContextKey)
	userID, valid := value.(int64)
	return userID, ok && valid && userID > 0
}

func chainIDFromPath(ctx *gin.Context) (int64, bool) {
	chainID, err := strconv.ParseInt(ctx.Param("chainId"), 10, 64)
	return chainID, err == nil && chainID > 0
}

func handleChainOperationError(ctx *gin.Context, operation string, err error) {
	var validationError *ValidationError
	switch {
	case errors.As(err, &validationError):
		httpserver.WriteError(ctx, http.StatusBadRequest, "validation_error", validationError.Error())
	case errors.Is(err, ErrChainNotFound):
		httpserver.WriteError(ctx, http.StatusNotFound, "chain_not_found", "chain not found")
	case errors.Is(err, ErrNotChainParticipant):
		httpserver.WriteError(ctx, http.StatusForbidden, "not_chain_participant", "user is not a participant of this chain")
	case errors.Is(err, ErrChainItemsUnavailable):
		httpserver.WriteError(ctx, http.StatusConflict, "chain_items_unavailable", "one or more chain items are no longer available")
	case errors.Is(err, ErrChainStatusConflict):
		httpserver.WriteError(ctx, http.StatusConflict, "chain_status_conflict", "chain status does not allow this operation")
	default:
		log.Printf("%s failed: %v", operation, err)
		httpserver.WriteError(ctx, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (h *Handler) DiscardChain(ctx *gin.Context) {

	httpserver.WriteError(ctx, http.StatusInternalServerError, "not_implemented", "not implemented")
}
