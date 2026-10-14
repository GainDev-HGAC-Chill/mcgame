// Open-sourced by BaoLT

// Game data RTMP handlers.
package gamedata

import (
	"context"

	appguild "mcgame-server/internal/application/guild"
	appitem "mcgame-server/internal/application/item"
	domainpet "mcgame-server/internal/domain/pet"
	gamedatamanager "mcgame-server/internal/gamedata"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type Handler struct {
	gameData     *gamedatamanager.Manager
	guildService *appguild.Service
	itemService  *appitem.Service
	petService   PetService
	logger       *zap.Logger
}

type PetService interface {
	GetPetDetail(ctx context.Context, charID int64, petID int64) (*domainpet.Pet, error)
}

func NewHandler(gameData *gamedatamanager.Manager, itemService *appitem.Service, logger *zap.Logger) *Handler {
	return &Handler{
		gameData:    gameData,
		itemService: itemService,
		logger:      logger,
	}
}

func (h *Handler) SetGuildService(guildService *appguild.Service) {
	h.guildService = guildService
}

func (h *Handler) SetPetService(petService PetService) {
	h.petService = petService
}

func (h *Handler) RegisterHandlers(dispatcher *rtmp.RPCDispatcher) {
	dispatcher.Register("gdc", h.GetDataConfig)
	dispatcher.Register("getDataPackageClient", h.GetDataPackageClient)
	dispatcher.Register("getItemInst_f", h.GetItemInstF)
}
