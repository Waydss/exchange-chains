package cb

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

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
// @Param user_id - int64
// @Param wished_item_id - int64
// @Success 202 {}
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

	h.service.EnqueueChainRequest(ChainRequest{User: &foundAccount, WishedItemId: request.WishedItemId})

	ctx.Status(http.StatusAccepted)
}

type CreateChainRequest struct {
	UserId       int64 `json:"user_id"`
	WishedItemId int64 `json:"wished_item_id"`
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
	User         *userCandidate `json:"user_id"`
	WishedItemId int64          `json:"wished_item"`
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

// ListChainsResponse is returned by the read-only chains endpoint.
type ListChainsResponse struct {
	Chains []ChainReadDTO `json:"chains"`
}

// ChainReadDTO is the read-only representation of a chain with its barters.
type ChainReadDTO struct {
	ID        int64           `json:"id"`
	RootID    int64           `json:"root_id"`
	RootWish  WishReadDTO     `json:"root_wish"`
	Signature string          `json:"signature"`
	Status    string          `json:"status"`
	Score     float64         `json:"score"`
	CreatedAt time.Time       `json:"created_at"`
	Barters   []BarterReadDTO `json:"barters"`
}

// BarterReadDTO is the read-only representation of one barter inside a chain.
type BarterReadDTO struct {
	ID       int64       `json:"id"`
	Position int64       `json:"position"`
	From     int64       `json:"from_user_id"`
	To       int64       `json:"to_user_id"`
	Give     GiveReadDTO `json:"give"`
	Wish     WishReadDTO `json:"wish"`
	Status   string      `json:"status"`
}

// GiveReadDTO is the read-only representation of an offered item.
type GiveReadDTO struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Title      string `json:"title"`
	CategoryID int64  `json:"category_id"`
	PriceMin   int64  `json:"price_min"`
	PriceMax   int64  `json:"price_max"`
	Status     string `json:"status"`
}

// WishReadDTO is the read-only representation of a wanted item.
type WishReadDTO struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	CategoryID    int64  `json:"category_id"`
	PriceMin      int64  `json:"price_min"`
	PriceMax      int64  `json:"price_max"`
	TitleKeywords string `json:"title_keywords"`
}

func newWishReadDTO(wish exchange.WishedItem) WishReadDTO {
	return WishReadDTO{
		ID:            wish.ID,
		UserID:        wish.UserID,
		CategoryID:    wish.CategoryID,
		PriceMin:      wish.PriceMin,
		PriceMax:      wish.PriceMax,
		TitleKeywords: wish.TitleKeywords,
	}
}

func newChainReadDTO(chain exchange.Chain) ChainReadDTO {
	barters := make([]BarterReadDTO, 0, len(chain.Barters))
	for _, barter := range chain.Barters {
		barters = append(barters, BarterReadDTO{
			ID:       barter.ID,
			Position: int64(len(barters) + 1),
			From:     barter.From,
			To:       barter.To,
			Give: GiveReadDTO{
				ID:         barter.Give.ID,
				UserID:     barter.Give.UserID,
				Title:      barter.Give.Title,
				CategoryID: barter.Give.CategoryID,
				PriceMin:   barter.Give.PriceMin,
				PriceMax:   barter.Give.PriceMax,
				Status:     string(barter.Give.Status),
			},
			Wish:   newWishReadDTO(barter.Wish),
			Status: string(barter.Status),
		})
	}

	return ChainReadDTO{
		ID:        chain.ID,
		RootID:    chain.RootID,
		RootWish:  newWishReadDTO(chain.RootWish),
		Signature: chain.Signature,
		Status:    string(chain.Status),
		Score:     chain.Score,
		CreatedAt: chain.CreatedAt,
		Barters:   barters,
	}
}
