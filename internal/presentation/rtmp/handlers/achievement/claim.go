// Open-sourced by BaoLT

// Achievement reward claim handler.
package achievement

import (
	"strconv"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (h *Handler) GetAchieveAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	if h.achievementService == nil || h.gameDataManager == nil {
		return map[string]interface{}{"flag": false}, nil
	}

	achievementID, err := parseAchievementID(args[0])
	if err != nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}
	template := h.gameDataManager.GetAchievement(int(achievementID))
	if template == nil || int(template.IsAward) <= 0 {
		return map[string]interface{}{"flag": false}, nil
	}

	token := uuid.New()
	if err := h.achievementService.ReserveClaim(ctx.Context, characterID, achievementID, token); err != nil {
		h.logClaimFailure(characterID, achievementID, "reserve", err)
		h.sendClaimFailure(ctx)
		return map[string]interface{}{"flag": false}, nil
	}
	if err := h.grantReward(ctx, characterID, achievementID); err != nil {
		releaseErr := h.achievementService.ReleaseClaim(ctx.Context, characterID, achievementID, token)
		h.logClaimFailure(characterID, achievementID, "grant", err)
		if releaseErr != nil {
			h.logClaimFailure(characterID, achievementID, "release", releaseErr)
		}
		h.sendClaimFailure(ctx)
		return map[string]interface{}{"flag": false}, nil
	}
	if err := h.achievementService.FinalizeClaim(ctx.Context, characterID, achievementID, token); err != nil {
		h.logger.Error("Achievement reward granted but claim finalization failed; reservation retained",
			zap.Int64("character_id", characterID),
			zap.Int64("achievement_id", achievementID),
			zap.String("claim_token", token.String()),
			zap.Error(err))
		h.sendClaimFailure(ctx)
		return map[string]interface{}{"flag": false}, nil
	}

	if ctx.Connection != nil {
		_ = ctx.Connection.SendCallback("updateAchieveAwardLog", map[string]interface{}{
			"aid": achievementID,
			"num": 1,
		})
	}
	return map[string]interface{}{"flag": true, "aid": achievementID}, nil
}

func parseAchievementID(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), nil
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	default:
		return 0, pkgerrors.ErrInvalidArgs
	}
}

func (h *Handler) logClaimFailure(characterID, achievementID int64, stage string, err error) {
	h.logger.Warn("Failed to claim achievement reward",
		zap.Int64("character_id", characterID),
		zap.Int64("achievement_id", achievementID),
		zap.String("stage", stage),
		zap.Error(err))
}

func (h *Handler) sendClaimFailure(ctx *rtmp.RPCContext) {
	if ctx != nil && ctx.Connection != nil {
		_ = ctx.Connection.SendCallback("onRedMsg", "Không thể nhận phần thưởng thành tựu.")
	}
}
