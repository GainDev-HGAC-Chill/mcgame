// Open-sourced by BaoLT

// Achievement reward claim handler.
package achievement

import (
	"encoding/json"
	"strconv"

	appachievement "mcgame-server/internal/application/achievement"
	domainachievement "mcgame-server/internal/domain/achievement"
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
	grant, err := h.achievementService.PrepareClaim(ctx.Context, characterID, achievementID, token)
	if err != nil {
		h.logClaimFailure(characterID, achievementID, "reserve", err)
		h.sendClaimFailure(ctx)
		return map[string]interface{}{"flag": false}, nil
	}
	if grant.Status == domainachievement.RewardGrantFinalized {
		return map[string]interface{}{"flag": true, "aid": achievementID}, nil
	}
	receipt := domainachievement.RewardReceipt{}
	if grant.Status == domainachievement.RewardGrantGranted {
		if err := json.Unmarshal(grant.Receipt, &receipt); err != nil {
			h.logClaimFailure(characterID, achievementID, "receipt", err)
			h.sendClaimFailure(ctx)
			return map[string]interface{}{"flag": false}, nil
		}
	} else {
		receipt, err = h.grantReward(ctx, characterID, achievementID)
		if err != nil {
			markErr := h.achievementService.MarkClaimFailed(ctx.Context, characterID, achievementID, token, err)
			h.logClaimFailure(characterID, achievementID, "grant", err)
			if markErr != nil {
				h.logClaimFailure(characterID, achievementID, "mark-failed", markErr)
			}
			h.sendClaimFailure(ctx)
			return map[string]interface{}{"flag": false}, nil
		}
		if err := h.achievementService.MarkClaimGranted(ctx.Context, characterID, achievementID, token, receipt); err != nil {
			h.logger.Error("Achievement reward delivered but receipt persistence failed; idempotent grant key retained",
				zap.Int64("character_id", characterID),
				zap.Int64("achievement_id", achievementID),
				zap.String("claim_token", token.String()),
				zap.Error(err))
			h.sendClaimFailure(ctx)
			return map[string]interface{}{"flag": false}, nil
		}
	}
	if err := appachievement.ValidateRewardReceipt(achievementID, receipt); err != nil {
		h.logClaimFailure(characterID, achievementID, "receipt-validation", err)
		h.sendClaimFailure(ctx)
		return map[string]interface{}{"flag": false}, nil
	}
	if err := h.achievementService.FinalizeClaim(ctx.Context, characterID, achievementID); err != nil {
		h.logger.Error("Achievement reward granted but claim finalization failed; grant receipt retained",
			zap.Int64("character_id", characterID),
			zap.Int64("achievement_id", achievementID),
			zap.String("claim_token", token.String()),
			zap.Error(err))
		h.sendClaimFailure(ctx)
		return map[string]interface{}{"flag": false}, nil
	}
	if err := h.sendRewardCallbacks(ctx, characterID, achievementID, receipt); err != nil {
		h.logClaimFailure(characterID, achievementID, "callback", err)
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
