// Open-sourced by BaoLT

// Inventory- and currency-consuming Monster Heart actions fail closed until one atomic ownership boundary exists.
package monsterheart

import (
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) OpenHoleMonsterHeart(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.rejectUnsupportedMutation(ctx, "openHoleMonsterHeart", args)
	return nil, nil
}

func (h *Handler) OpenHoleMonsterHeartJingang(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.rejectUnsupportedMutation(ctx, "openHoleMonsterHeartJingang", args)
	return nil, nil
}

func (h *Handler) UpgradeMonsterHeartItem(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.rejectUnsupportedMutation(ctx, "upgradeMonsterHeartItem", args)
	return nil, nil
}

func (h *Handler) ResolveMonsterHeart(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.rejectUnsupportedMutation(ctx, "resolveMonsterHeart", args)
	return nil, nil
}

func (h *Handler) UpgradeMonsterHeartBox(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.rejectUnsupportedMutation(ctx, "upgradeMonsterHeartBox", args)
	if ctx == nil || ctx.Connection == nil {
		return nil, nil
	}
	charID, ok := h.charIDFromCtx(ctx)
	if !ok || h.service == nil {
		return nil, nil
	}
	state, err := h.service.Load(ctx.Context, charID)
	if err == nil && state != nil {
		levels := stateToWire(state)["data"].(map[string]interface{})["heartBoxLev"]
		_ = ctx.Connection.SendCallback("onUpgradeMonsterHeartBox", levels)
	}
	return nil, nil
}

func (h *Handler) rejectUnsupportedMutation(ctx *rtmp.RPCContext, method string, args []interface{}) {
	fields := []zap.Field{zap.String("method", method), zap.Any("args", args)}
	if ctx != nil {
		fields = append(fields, zap.String("character_id", ctx.CharacterID), zap.Uint32("conn_id", ctx.ConnID))
	}
	h.logger.Warn("Monster Heart mutation blocked: atomic ownership model unavailable", fields...)
	h.sendCurrentPanel(ctx)
}
