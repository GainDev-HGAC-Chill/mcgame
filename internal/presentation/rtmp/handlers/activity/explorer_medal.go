// Open-sourced by BaoLT

// Explorer medal handlers expose point-funded and gold-funded progression.
package activity

import (
	"errors"

	appexplorermedal "mcgame-server/internal/application/explorermedal"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) InitEMPanel(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "InitEMPanel")
	if err != nil {
		return nil, err
	}

	if h.explorerMedalSvc == nil {
		h.logger.Warn("InitEMPanel: explorerMedalSvc not configured",
			zap.Uint32("conn_id", ctx.ConnID))
		return []interface{}{0, map[string]interface{}{}, 0, 0}, nil
	}

	reqCtx := ctx.Context
	response, err := h.explorerMedalSvc.GetPanelResponse(reqCtx, characterID)
	if err != nil {
		h.logger.Warn("InitEMPanel: failed to load explorer medal state",
			zap.Uint32("conn_id", ctx.ConnID),
			zap.Int64("character_id", characterID),
			zap.Error(err))
		return []interface{}{0, map[string]interface{}{}, 0, 0}, nil
	}

	h.logger.Info("InitEMPanel: returning panel state",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	return response, nil
}

func (h *Handler) LevelUpEMedal(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "LevelUpEMedal")
	if err != nil {
		return nil, err
	}

	h.logger.Info("LevelUpEMedal called",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	if h.explorerMedalSvc == nil {
		h.logger.Warn("LevelUpEMedal: explorerMedalSvc not configured",
			zap.Uint32("conn_id", ctx.ConnID))
		return nil, nil
	}

	reqCtx := ctx.Context
	result, err := h.explorerMedalSvc.LevelUp(reqCtx, characterID)
	if err != nil {
		if errors.Is(err, appexplorermedal.ErrAtMaxLevel) {
			h.logger.Info("LevelUpEMedal: already at max level",
				zap.Uint32("conn_id", ctx.ConnID),
				zap.Int64("character_id", characterID))
			h.sendActivityCallback(ctx, "onSystemSay", "Đã đạt Huy Chương cao cấp nhất")
			return nil, nil
		}
		if errors.Is(err, appexplorermedal.ErrInsufficientScore) {
			h.logger.Info("LevelUpEMedal: insufficient EMP score",
				zap.Uint32("conn_id", ctx.ConnID),
				zap.Int64("character_id", characterID))
			h.sendActivityCallback(ctx, "onSystemSay", "Không đủ điểm để nâng cấp Huy Chương")
			return nil, nil
		}
		h.logger.Error("LevelUpEMedal: level up failed",
			zap.Uint32("conn_id", ctx.ConnID),
			zap.Int64("character_id", characterID),
			zap.Error(err))
		return nil, nil
	}

	h.sendExplorerMedalResult(ctx, characterID, result)

	h.logger.Info("LevelUpEMedal: level up successful",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID),
		zap.Int("new_level", result.Level),
		zap.Int("progress", result.Progress))

	return nil, nil
}

func (h *Handler) LevelUpEMedalByGold(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "LevelUpEMedalByGold")
	if err != nil {
		return nil, err
	}
	if h.explorerMedalSvc == nil || len(args) < 2 {
		return nil, nil
	}
	upgradeType, ok := parseIntArg(args[1])
	if !ok {
		return nil, nil
	}
	result, err := h.explorerMedalSvc.LevelUpByGold(ctx.Context, characterID, upgradeType)
	if err != nil {
		switch {
		case errors.Is(err, appexplorermedal.ErrAtMaxLevel):
			h.sendActivityCallback(ctx, "onSystemSay", "Đã đạt Huy Chương cao cấp nhất")
		case errors.Is(err, appexplorermedal.ErrInsufficientGold):
			h.sendActivityCallback(ctx, "onSystemSay", "Không đủ vàng")
		default:
			h.logger.Error("LevelUpEMedalByGold failed", zap.Int64("character_id", characterID), zap.Error(err))
		}
		return nil, nil
	}
	h.sendExplorerMedalResult(ctx, characterID, result)
	return nil, nil
}

func (h *Handler) sendExplorerMedalResult(ctx *rtmp.RPCContext, characterID int64, result appexplorermedal.LevelUpResult) {
	h.sendActivityCallback(ctx, "updateEMPanel", []interface{}{result.Level, result.PropMap, result.ExplorerPnt, result.Progress})
	if result.WalletSpent > 0 {
		h.sendActivityCallback(ctx, "onAddMoney", float64(characterID), "explorerPnt", float64(-result.WalletSpent), float64(result.ExplorerPnt))
	}
	if result.GoldSpent > 0 {
		h.sendActivityCallback(ctx, "onAddMoney", float64(characterID), "gold", float64(-result.GoldSpent), float64(result.Gold))
		h.sendActivityCallback(ctx, "onUPP", map[string]interface{}{"gold": result.Gold})
	}
}
