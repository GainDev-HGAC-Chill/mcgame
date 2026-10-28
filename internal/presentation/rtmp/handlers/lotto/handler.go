// Open-sourced by BaoLT

// Lottery bag RPC handlers. Three parallel bags (lotto / lottery / luckDraw) each expose
// get / getSingle(claim one) / getAll(claim all) / throwAll / sort, all of which push the
// updated bag via the bag's update* callback. The Wish-Tree draw RPCs getLottoData
// (returns the per-type display grid of TBL_PLAN ids) and lottoByClient (rolls the pool,
// deposits winnings, pushes updateLottoBag) are also registered here.
package lotto

import (
	"errors"
	"strconv"
	"time"

	appachievement "mcgame-server/internal/application/achievement"
	applotto "mcgame-server/internal/application/lotto"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"

	"go.uber.org/zap"
)

type Handler struct {
	service        *applotto.Service
	achievements   *appachievement.Service
	itemDTOBuilder itemDTOBuilder
	logger         *zap.Logger
}

func (h *Handler) SetAchievementService(service *appachievement.Service) {
	h.achievements = service
}

type itemDTOBuilder interface {
	BuildClientItemDTO(item *domainitem.Item) map[string]interface{}
}

type callbackSender interface {
	SendCallback(method string, args ...interface{}) error
}

func NewHandler(service *applotto.Service, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{service: service, logger: logger}
}

func (h *Handler) SetItemDTOBuilder(builder itemDTOBuilder) {
	h.itemDTOBuilder = builder
}

type bagRoutes struct {
	kind     string
	callback string
	get      string
	single   string
	all      string
	throw    string
	sort     string
}

func (h *Handler) RegisterHandlers(d *rtmp.RPCDispatcher) {
	routes := []bagRoutes{
		{applotto.BagLotto, "updateLottoBag", "getLottoBag", "getSingleItem", "getAllItem", "throwAllItem", "lottoBagSort"},
		{applotto.BagLottery, "updateLotteryBag", "getLotteryBag", "getSingleLotteryItem", "getAllLotteryItem", "throwAllLotteryItem", "lotteryBagSort"},
		{applotto.BagLuckDraw, "updateLuckDrawBag", "getLuckDrawBag", "getSingleLuckDrawItem", "getAllLuckDrawItem", "throwAllLuckDrawItem", "luckDrawBagSort"},
	}
	for _, r := range routes {
		d.Register(r.get, h.makeGet(r.kind, r.callback))
		d.Register(r.single, h.makeSingle(r.kind, r.callback))
		d.Register(r.all, h.makeAll(r.kind, r.callback))
		d.Register(r.throw, h.makeThrow(r.kind, r.callback))
		d.Register(r.sort, h.makeSort(r.kind, r.callback))
	}
	d.Register("getLottoData", h.getLottoData)
	d.Register("lottoByClient", h.lottoByClient)
}

func (h *Handler) getLottoData(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := charIDFromCtx(ctx)
	if !ok {
		return nil, nil
	}
	data, err := h.service.GetLottoData(ctx.Context, charID)
	if err != nil {
		h.logger.Error("getLottoData failed", zap.Error(err))
		return map[string]interface{}{"lottoAward": []interface{}{}, "highestAwardArr": []interface{}{}}, nil
	}
	return data, nil
}

func (h *Handler) lottoByClient(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := charIDFromCtx(ctx)
	if !ok {
		return false, nil
	}
	drawType, num := 0, 0
	if len(args) > 0 {
		if m, ok := args[0].(map[string]interface{}); ok {
			if n, err := parseIntArg(m["type"]); err == nil {
				drawType = n
			}
			if n, err := parseIntArg(m["num"]); err == nil {
				num = n
			}
		}
	}
	bag, err := h.service.Draw(ctx.Context, charID, drawType, num)
	if err != nil {
		h.logger.Warn("lottoByClient draw failed",
			zap.Int("type", drawType), zap.Int("num", num), zap.Error(err))
		return false, nil
	}
	// Đồng bộ vàng ngay sau khi quay thưởng thành công.
	newGold, goldErr := h.service.GetGold(ctx.Context, charID)
	if goldErr != nil {
		h.logger.Error("lotto: failed to refresh gold",
			zap.Int64("character_id", charID),
			zap.Error(goldErr))
	} else {
		if err := ctx.Connection.SendCallback("onUPP", map[string]interface{}{
			"gold": newGold,
		}); err != nil {
			h.logger.Warn("lotto: failed to send gold update", zap.Error(err))
		}
	}

	// Cập nhật Kho Ước Nguyện.
	_ = ctx.Connection.SendCallback("updateLottoBag", bag)
	if h.achievements != nil {
		update, progressErr := h.achievements.ProcessPendingFacts(ctx.Context, charID, time.Now().UTC())
		if progressErr != nil {
			h.logger.Warn("lotto: achievement fact processing failed", zap.Int64("character_id", charID), zap.Error(progressErr))
		} else {
			rtmputils.SendAchievementProgressCallbacks(ctx.Connection, update)
		}
	}

	return true, nil
}

func (h *Handler) makeGet(kind, callback string) rtmp.HandlerFunc {
	return func(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
		charID, ok := charIDFromCtx(ctx)
		if !ok {
			return nil, nil
		}
		bag, err := h.service.GetBag(ctx.Context, charID, kind)
		if err != nil {
			h.logger.Error("lotto get bag failed", zap.String("kind", kind), zap.Error(err))
			return nil, nil
		}
		_ = ctx.Connection.SendCallback(callback, bag)
		return bag, nil
	}
}

func (h *Handler) makeSingle(kind, callback string) rtmp.HandlerFunc {
	return func(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
		charID, ok := charIDFromCtx(ctx)
		if !ok {
			return nil, nil
		}
		index := 0
		if len(args) > 0 {
			if n, err := parseIntArg(args[0]); err == nil {
				index = n
			}
		}
		result, err := h.service.ClaimSingle(ctx.Context, charID, kind, index)
		if err != nil {
			return rtmp.ErrorToResponse(err), nil
		}
		h.sendGrantedItems(ctx.Connection, result.GrantedItems)
		_ = ctx.Connection.SendCallback(callback, result.Bag)
		return nil, nil
	}
}

func (h *Handler) makeAll(kind, callback string) rtmp.HandlerFunc {
	return func(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
		charID, ok := charIDFromCtx(ctx)
		if !ok {
			return nil, nil
		}
		result, err := h.service.ClaimAll(ctx.Context, charID, kind)
		if err != nil {
			return rtmp.ErrorToResponse(err), nil
		}
		h.sendGrantedItems(ctx.Connection, result.GrantedItems)
		_ = ctx.Connection.SendCallback(callback, result.Bag)
		return nil, nil
	}
}

func (h *Handler) sendGrantedItems(sender callbackSender, items []*domainitem.Item) {
	if sender == nil || h.itemDTOBuilder == nil {
		return
	}
	byID := make(map[int64]*domainitem.Item, len(items))
	order := make([]int64, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if _, exists := byID[item.ID]; !exists {
			order = append(order, item.ID)
		}
		byID[item.ID] = item
	}
	for _, id := range order {
		_ = sender.SendCallback("onAddCharactorSlot", h.itemDTOBuilder.BuildClientItemDTO(byID[id]))
	}
}

func (h *Handler) makeThrow(kind, callback string) rtmp.HandlerFunc {
	return func(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
		charID, ok := charIDFromCtx(ctx)
		if !ok {
			return nil, nil
		}
		bag, err := h.service.ThrowAll(ctx.Context, charID, kind)
		if err != nil {
			return rtmp.ErrorToResponse(err), nil
		}
		_ = ctx.Connection.SendCallback(callback, bag)
		return nil, nil
	}
}

func (h *Handler) makeSort(kind, callback string) rtmp.HandlerFunc {
	return func(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
		charID, ok := charIDFromCtx(ctx)
		if !ok {
			return nil, nil
		}
		bag, err := h.service.Sort(ctx.Context, charID, kind)
		if err != nil {
			return rtmp.ErrorToResponse(err), nil
		}
		_ = ctx.Connection.SendCallback(callback, bag)
		return nil, nil
	}
}

func parseIntArg(value any) (int, error) {
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case string:
		return strconv.Atoi(typed)
	}
	return 0, errors.New("unsupported argument type")
}

func charIDFromCtx(ctx *rtmp.RPCContext) (int64, bool) {
	if ctx == nil || ctx.CharacterID == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
