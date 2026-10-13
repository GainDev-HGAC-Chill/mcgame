// Open-sourced by BaoLT

// Activity award handlers cover consume, point, shop, gift, and event reward views.
package activity

import (
	"time"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func (h *Handler) GetStageConsumeAwardList(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "GetStageConsumeAwardList")
	if err != nil {
		return nil, err
	}

	h.logger.Debug("GetStageConsumeAwardList: fetching stage consume award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	stageData := map[string]interface{}{"stageNum": 0}
	consumptionTotals := map[string]interface{}{}
	h.sendActivityCallback(ctx, "onGetStageConsumeAwardList", stageData, consumptionTotals)
	response := []interface{}{stageData, consumptionTotals}

	h.logger.Info("GetStageConsumeAwardList: returning stage consume award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	return response, nil
}

func (h *Handler) GetConsumeAwardList(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "GetConsumeAwardList")
	if err != nil {
		return nil, err
	}

	h.logger.Debug("GetConsumeAwardList: fetching consume award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	awardStages := map[string]interface{}{}
	h.sendActivityCallback(ctx, "onGetConsumeAwardList", awardStages, 0)
	response := []interface{}{awardStages, 0}

	h.logger.Info("GetConsumeAwardList: returning consume award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	return response, nil
}

func (h *Handler) GetPointAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "GetPointAward")
	if err != nil {
		return nil, err
	}

	h.logger.Debug("GetPointAward: fetching point award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	response := map[string]interface{}{}

	h.logger.Info("GetPointAward: returning point award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	return response, nil
}

func (h *Handler) SearchGameGift(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "SearchGameGift")
	if err != nil {
		return nil, err
	}

	h.logger.Debug("SearchGameGift: searching game gifts",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	h.sendActivityCallback(ctx, "onGetItemIdAward", "[]")

	h.logger.Info("SearchGameGift: returning game gifts",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID),
		zap.Int("gift_count", 0))

	return "[]", nil
}

func (h *Handler) GetShopAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "GetShopAward")
	if err != nil {
		return nil, err
	}

	h.logger.Debug("GetShopAward: fetching shop award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	response := map[string]interface{}{}

	h.logger.Info("GetShopAward: returning shop award list",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID))

	return response, nil
}

func (h *Handler) CheckDoubleExpTime(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charIDStr := ctx.CharacterID
	if charIDStr == "" {
		h.logger.Warn("CheckDoubleExpTime: no character ID in session", zap.Uint32("conn_id", ctx.ConnID))
		return nil, pkgerrors.ErrUnauthorized
	}

	now := time.Now()
	isDoubleExpActive := now.Hour() >= 18 && now.Hour() < 22
	if isDoubleExpActive {
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, now.Location())
		if now.After(endTime) {
			endTime = endTime.Add(24 * time.Hour)
		}
		timeLeftSeconds := int64(endTime.Sub(now).Seconds())
		buffData := map[string]interface{}{
			"bid":      1001,
			"id":       0,
			"type":     0,
			"timeLeft": timeLeftSeconds,
			"desc":     "Double Experience Event Active",
		}

		h.logger.Info("CheckDoubleExpTime: double exp active",
			zap.Uint32("conn_id", ctx.ConnID),
			zap.Int64("time_left_seconds", timeLeftSeconds))

		if err := ctx.Connection.SendCallback("addDoubleExpBuff", buffData); err != nil {
			h.logger.Warn("Failed to send addDoubleExpBuff callback",
				zap.Uint32("conn_id", ctx.ConnID),
				zap.Error(err))
		}
	} else {
		if err := ctx.Connection.SendCallback("endDoubleExpBuff"); err != nil {
			h.logger.Warn("Failed to send endDoubleExpBuff callback",
				zap.Uint32("conn_id", ctx.ConnID),
				zap.Error(err))
		}
	}

	return nil, nil
}
