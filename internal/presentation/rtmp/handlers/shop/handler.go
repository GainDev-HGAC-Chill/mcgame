// Open-sourced by BaoLT

// Shop handler core: dependency wiring and registration.
package shop

import (
	appbloodbag "mcgame-server/internal/application/bloodbag"
	appquest "mcgame-server/internal/application/quest"
	appshop "mcgame-server/internal/application/shop"
	domainauth "mcgame-server/internal/domain/auth"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/infrastructure/rtmp"
	questhandler "mcgame-server/internal/presentation/rtmp/handlers/quest"

	"go.uber.org/zap"
)

type Handler struct {
	logger       *zap.Logger
	shopService  appshop.ShopService
	questService *appquest.Service
	questNotify  *questhandler.QuestNotifyHelper
	accountRepo  domainauth.AccountRepository
	gameData     *gamedata.Manager
	bloodBag     *appbloodbag.Service
}

func NewHandler(logger *zap.Logger, shopService appshop.ShopService) *Handler {
	return &Handler{
		logger:      logger,
		shopService: shopService,
	}
}

func (h *Handler) SetQuestService(qs *appquest.Service) {
	h.questService = qs
}

func (h *Handler) SetQuestNotifyHelper(qnh *questhandler.QuestNotifyHelper) {
	h.questNotify = qnh
}

func (h *Handler) SetAccountRepository(accountRepo domainauth.AccountRepository) {
	h.accountRepo = accountRepo
}

func (h *Handler) SetGameDataManager(gameData *gamedata.Manager) {
	h.gameData = gameData
}

func (h *Handler) SetBloodBagService(service *appbloodbag.Service) {
	h.bloodBag = service
}

func (h *Handler) RegisterHandlers(dispatcher *rtmp.RPCDispatcher) {
	dispatcher.Register("getShopConfig", h.GetShopConfig)
	dispatcher.Register("getVipShopConfig", h.GetVipShopConfig)
	dispatcher.Register("getVipShopCharConfig", h.GetVipShopCharConfig)
	dispatcher.Register("getLimitShopConfig", h.GetLimitShopConfig)
	dispatcher.Register("getRemainShopConfig", h.GetRemainShopConfig)
	dispatcher.Register("buyAmountItemClient", h.BuyAmountItemClient)
	dispatcher.Register("buyItemClient", h.BuyItemClient)
	dispatcher.Register("buyVipShopItem", h.BuyVipShopItem)
	dispatcher.Register("buySystemItemClient", h.BuySystemItemClient)
	dispatcher.Register("sellItem", h.SellItem)
	dispatcher.Register("exchangeItem", h.ExchangeItem)
	dispatcher.Register("repairAll", h.RepairAll)
	dispatcher.Register("shopClosePanel", h.ShopClosePanel)
	dispatcher.Register("unlockMoney", h.UnlockMoney)
	dispatcher.Register("updateLimit", h.UpdateLimit)
	dispatcher.Register("getShopAwardStr", h.GetShopAwardStr)
	dispatcher.Register("changeMoneyType", h.ChangeMoneyType)
	dispatcher.Register("buySystemItemMulti", h.BuySystemItemMulti)
	dispatcher.Register("openShopDirect", h.OpenShopDirect)
	dispatcher.Register("clickShop", h.ClickShop)
	dispatcher.Register("buyBloodBag", h.BuyBloodBag)
}

func (h *Handler) refreshQuestProgress(ctx *rtmp.RPCContext, characterID int64) {
	if h.questNotify != nil {
		h.questNotify.RefreshQuestProgressSilently(ctx, characterID)
	}
}
