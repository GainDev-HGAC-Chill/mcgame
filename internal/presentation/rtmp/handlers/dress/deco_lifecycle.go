// Open-sourced by BaoLT

// Hồn Khí ownership, equip, visibility, visual-tier, and expiry RPC handlers.
package dress

import (
	"go.uber.org/zap"

	appdress "mcgame-server/internal/application/dress"
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) InitDecoratePanel(ctx *rtmp.RPCContext, _ []interface{}) (interface{}, error) {
	charID, ok := h.characterID(ctx)
	if !ok || h.decoHoleService == nil {
		return nil, nil
	}
	state, due, err := h.decoHoleService.RefreshExpiredDecoShows(ctx.Context, charID)
	if err != nil {
		h.logger.Error("initDecoratePanel: refresh failed", zap.Int64("character_id", charID), zap.Error(err))
		return nil, nil
	}
	if ctx.Connection != nil {
		_ = ctx.Connection.SendCallback("initDecoShowTimer", due)
	}
	return appdress.DecoHoleStateToWire(charID, state), nil
}

func (h *Handler) ActiveDecoShow(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, showID, position, ok := h.decoShowArgs(ctx, args)
	if !ok {
		return nil, nil
	}
	result, err := h.decoHoleService.ActivateDecoShow(ctx.Context, charID, showID, position)
	if err != nil {
		h.logDecoClientOrServiceError("activeDecoShow", charID, err)
		return nil, nil
	}
	if result.SpentGold > 0 && ctx.Connection != nil {
		_ = ctx.Connection.SendCallback("onAddMoney", float64(charID), "gold", float64(-result.SpentGold), float64(result.GoldAfter))
	}
	h.refreshCharacterBonuses(ctx, charID)
	return appdress.DecoHoleStateToWire(charID, result.State), nil
}

func (h *Handler) EquipDecoShow(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, showID, position, ok := h.decoShowArgs(ctx, args)
	if !ok {
		return nil, nil
	}
	state, err := h.decoHoleService.EquipDecoShow(ctx.Context, charID, showID, position)
	if err != nil {
		h.logDecoClientOrServiceError("equipDecoShow", charID, err)
		return nil, nil
	}
	h.refreshCharacterBonuses(ctx, charID)
	return appdress.DecoHoleStateToWire(charID, state), nil
}

func (h *Handler) EquipOffDecoShow(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, showID, position, ok := h.decoShowArgs(ctx, args)
	if !ok {
		return nil, nil
	}
	state, err := h.decoHoleService.EquipOffDecoShow(ctx.Context, charID, showID, position)
	if err != nil {
		h.logDecoClientOrServiceError("equipOffDecoShow", charID, err)
		return nil, nil
	}
	h.refreshCharacterBonuses(ctx, charID)
	return appdress.DecoHoleStateToWire(charID, state), nil
}

func (h *Handler) HideDecoShow(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.setDecoShowVisible(ctx, args, false)
}

func (h *Handler) ShowDecoShow(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.setDecoShowVisible(ctx, args, true)
}

func (h *Handler) ChangeShowLevel(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, ok := h.characterID(ctx)
	if !ok || h.decoHoleService == nil || len(args) < 2 {
		return nil, nil
	}
	showLevel, levelOK := toInt64(args[0])
	position, positionOK := toInt64(args[1])
	if !levelOK || !positionOK {
		return nil, nil
	}
	state, err := h.decoHoleService.ChangeDecoShowLevel(ctx.Context, charID, int(showLevel), int(position))
	if err != nil {
		h.logDecoClientOrServiceError("changeShowLvl", charID, err)
		return nil, nil
	}
	return appdress.DecoHoleStateToWire(charID, state), nil
}

func (h *Handler) UpdateActiveDecoShow(ctx *rtmp.RPCContext, _ []interface{}) (interface{}, error) {
	charID, ok := h.characterID(ctx)
	if !ok || h.decoHoleService == nil {
		return nil, nil
	}
	state, _, err := h.decoHoleService.RefreshExpiredDecoShows(ctx.Context, charID)
	if err != nil {
		h.logger.Error("updateActiveDecoShow: refresh failed", zap.Int64("character_id", charID), zap.Error(err))
		return nil, nil
	}
	h.refreshCharacterBonuses(ctx, charID)
	return appdress.DecoHoleStateToWire(charID, state), nil
}

func (h *Handler) decoShowArgs(ctx *rtmp.RPCContext, args []interface{}) (int64, int, int, bool) {
	charID, ok := h.characterID(ctx)
	if !ok || h.decoHoleService == nil || len(args) < 2 {
		return 0, 0, 0, false
	}
	showID, showOK := toInt64(args[0])
	position, positionOK := toInt64(args[1])
	if !showOK || !positionOK {
		return 0, 0, 0, false
	}
	return charID, int(showID), int(position), true
}

func (h *Handler) setDecoShowVisible(ctx *rtmp.RPCContext, args []interface{}, visible bool) (interface{}, error) {
	charID, ok := h.characterID(ctx)
	if !ok || h.decoHoleService == nil || len(args) < 1 {
		return nil, nil
	}
	position, positionOK := toInt64(args[0])
	if !positionOK {
		return nil, nil
	}
	state, err := h.decoHoleService.SetDecoShowVisible(ctx.Context, charID, int(position), visible)
	if err != nil {
		h.logDecoClientOrServiceError("setDecoShowVisible", charID, err)
		return nil, nil
	}
	return appdress.DecoHoleStateToWire(charID, state), nil
}

func (h *Handler) logDecoClientOrServiceError(method string, charID int64, err error) {
	if err == nil {
		return
	}
	switch err {
	case appdress.ErrDecoShowInvalid, appdress.ErrDecoShowNotOwned, appdress.ErrDecoShowNotEquipped, appdress.ErrDecoShowAlreadyOwned, appdress.ErrDecoShowInsufficientGold, appdress.ErrDecoShowInvalidLevel, appdress.ErrDecoHoleInvalidPosition:
		h.logger.Debug(method+": rejected", zap.Int64("character_id", charID), zap.Error(err))
	default:
		h.logger.Error(method+": failed", zap.Int64("character_id", charID), zap.Error(err))
	}
}
