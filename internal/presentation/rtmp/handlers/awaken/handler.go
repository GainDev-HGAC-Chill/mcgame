// Open-sourced by BaoLT

// Awakening ("Giác Tỉnh") RPC handler. addAwakenPoint / reduceAwakenPoint reply via
// the inline Responder result {awakenPoint, awakenPointUsed, awakenDict} (the client
// routes it to onAddAwakenPoint). awakening(autoBuy) / ensureBuyAwaken() drive the
// probabilistic level-up gacha and reply via the inline Responder onAwakening with one
// of three shapes: {needGold} | {success:true, awakenLevel, awakenPoint, awakenAdd} |
// {success:false, itemNum, awakenAdd}. See internal/application/awaken/gacha.go.
package awaken

import (
	"context"
	"strconv"

	appawaken "mcgame-server/internal/application/awaken"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type Handler struct {
	service       *appawaken.Service
	characterRepo characterStatLoader
	bonusProvider characterBonusProvider
	logger        *zap.Logger
}

type characterStatLoader interface {
	FindByID(ctx context.Context, id int64) (*domainchar.Character, error)
}

type characterBonusProvider interface {
	AggregateEquipmentStats(ctx context.Context, charID int64) domainchar.EquipmentStatBonuses
}

type callbackSender interface {
	SendCallback(method string, args ...interface{}) error
}

func NewHandler(service *appawaken.Service, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{service: service, logger: logger}
}

func (h *Handler) SetStatRefreshDependencies(characterRepo characterStatLoader, bonusProvider characterBonusProvider) {
	h.characterRepo = characterRepo
	h.bonusProvider = bonusProvider
}

func (h *Handler) RegisterHandlers(d *rtmp.RPCDispatcher) {
	d.Register("addAwakenPoint", h.AddPoint)
	d.Register("reduceAwakenPoint", h.ReducePoint)
	d.Register("awakening", h.Awakening)
	d.Register("ensureBuyAwaken", h.EnsureBuyAwaken)
}

func (h *Handler) Awakening(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := charIDFromCtx(ctx)
	if !ok {
		return nil, nil
	}
	autoBuy := boolArg(args, 0)
	data, err := h.service.Awaken(ctx.Context, charID, autoBuy)
	if err != nil {
		h.logger.Warn("awakening failed", zap.Int64("char_id", charID), zap.Error(err))
		return nil, nil
	}
	if awakeningSucceeded(data) {
		h.sendStatRefresh(ctx.Context, ctx.Connection, charID)
	}
	return data, nil
}

func (h *Handler) EnsureBuyAwaken(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := charIDFromCtx(ctx)
	if !ok {
		return nil, nil
	}
	data, err := h.service.EnsureBuyAwaken(ctx.Context, charID)
	if err != nil {
		h.logger.Warn("ensureBuyAwaken failed", zap.Int64("char_id", charID), zap.Error(err))
		return nil, nil
	}
	if awakeningSucceeded(data) {
		h.sendStatRefresh(ctx.Context, ctx.Connection, charID)
	}
	return data, nil
}

func (h *Handler) AddPoint(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := charIDFromCtx(ctx)
	if !ok {
		return nil, nil
	}
	skillID, ok := intArg(args, 0)
	if !ok {
		return nil, nil
	}
	data, err := h.service.AddPoint(ctx.Context, charID, skillID)
	if err != nil {
		h.logger.Warn("addAwakenPoint failed", zap.Int64("char_id", charID), zap.Error(err))
		return nil, nil
	}
	return data, nil
}

func (h *Handler) ReducePoint(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := charIDFromCtx(ctx)
	if !ok {
		return nil, nil
	}
	skillID, ok := intArg(args, 0)
	if !ok {
		return nil, nil
	}
	data, err := h.service.ReducePoint(ctx.Context, charID, skillID)
	if err != nil {
		h.logger.Warn("reduceAwakenPoint failed", zap.Int64("char_id", charID), zap.Error(err))
		return nil, nil
	}
	return data, nil
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

func boolArg(args []interface{}, idx int) bool {
	if idx < 0 || idx >= len(args) {
		return false
	}
	switch v := args[idx].(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case string:
		return v == "true" || v == "1"
	}
	return false
}

func intArg(args []interface{}, idx int) (int, bool) {
	if idx < 0 || idx >= len(args) {
		return 0, false
	}
	switch v := args[idx].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

func awakeningSucceeded(data map[string]interface{}) bool {
	success, ok := data["success"].(bool)
	return ok && success
}

func (h *Handler) sendStatRefresh(ctx context.Context, sender callbackSender, charID int64) {
	if h == nil || h.characterRepo == nil || h.bonusProvider == nil || sender == nil {
		return
	}
	char, err := h.characterRepo.FindByID(ctx, charID)
	if err != nil || char == nil {
		h.logger.Warn("awakening stat refresh failed", zap.Int64("char_id", charID), zap.Error(err))
		return
	}
	bonuses := h.bonusProvider.AggregateEquipmentStats(ctx, charID)
	_ = sender.SendCallback("onUPP", domainchar.BuildUPPPayloadWithEquipment(char, bonuses))
}
