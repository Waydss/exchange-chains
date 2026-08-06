package cb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/L-ka080/DDB-6-case/internal/chain_builder/entities"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateChainsForItem godoc
//
// @Summary Прямой способ создания цепочки, не завязанный на состоянии предметов в системе
// @Description Создание цепочки на основе переданного UserId и WishedItemId из БД
// @Tags chains
// @Produce Status Code
// @Security BearerAuth
// @Success 200 {plain/text}
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 404 {object} httpserver.ErrorResponse
// @Router /api/v1/chain [post]
func CreateChainsForItem(ctx *gin.Context) {
	// FIXME I think this function will be changed completely to fit current server's structure...

	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()

	request := CreateChainRequest{}
	decoder.Decode(&request)

	if request.UserId == 0 {
		ctx.Status(http.StatusBadRequest)
		ctx.Error(fmt.Errorf("Request body is not valid"))
		return
	}

	// TODO Add here WishedItem validation too

	context := context.Background()

	// FIXME We need to start using gorm, because i can't live without it...
	foundAccount, err := gorm.G[entities.Account](db.DB).
		Where("id = ?", request.UserId).
		Preload("Wishes", nil).
		Preload("Gives", nil).
		First(context)

	if err != nil {

		// FIXME This must be changed to match the ErrorResponse schema
		ctx.Status(http.StatusBadRequest)
		ctx.Error(fmt.Errorf("User not found"))
		return
	}

	ChainRequestQuene <- ChainRequest{UserAccount: &foundAccount, WishedItemId: request.WishedItemId}

	ctx.Status(http.StatusOK)
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
