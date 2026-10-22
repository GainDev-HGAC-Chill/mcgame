package activity

import (
	"errors"

	appitem "mcgame-server/internal/application/item"
	appspecialtraining "mcgame-server/internal/application/specialtraining"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) GetTXKCData(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, _, err := h.loadActivityCharacter(ctx, "GetTXKCData")
	if err != nil {
		return nil, err
	}
	if h.specialTrainingSvc == nil {
		return nil, nil
	}
	payload, err := h.specialTrainingSvc.GetData(ctx.Context, characterID)
	if err != nil {
		h.handleTXKCError(ctx, "GetTXKCData", characterID, err)
		return nil, nil
	}
	h.sendActivityCallback(ctx, "onTXKCGetData", payload)
	return payload, nil
}

func (h *Handler) OpenTXKCAdvanceCourse(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.mutateTXKCGold(ctx, "OpenTXKCAdvanceCourse", func(characterID int64) (appspecialtraining.MutationResult, error) {
		return h.specialTrainingSvc.OpenPremium(ctx.Context, characterID)
	})
}

func (h *Handler) BuyTXKCEXP(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}
	count, ok := parseIntArg(args[0])
	if !ok {
		return nil, nil
	}
	return h.mutateTXKCGold(ctx, "BuyTXKCEXP", func(characterID int64) (appspecialtraining.MutationResult, error) {
		return h.specialTrainingSvc.BuyEXP(ctx.Context, characterID, count)
	})
}

func (h *Handler) TakeTXKCDailyQuest(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.mutateTXKCState(ctx, "TakeTXKCDailyQuest", func(characterID int64) (map[string]interface{}, error) {
		return h.specialTrainingSvc.OpenDailyTasks(ctx.Context, characterID)
	})
}

func (h *Handler) TXKCGetProgressEXP(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}
	taskType, okType := parseIntArg(args[0])
	taskID, okID := parseIntArg(args[1])
	if !okType || !okID {
		return nil, nil
	}
	return h.mutateTXKCState(ctx, "TXKCGetProgressEXP", func(characterID int64) (map[string]interface{}, error) {
		return h.specialTrainingSvc.ClaimTaskEXP(ctx.Context, characterID, taskType, taskID)
	})
}

func (h *Handler) TakeTXKCFreeAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.takeTXKCReward(ctx, false)
}

func (h *Handler) TakeTXKCAdvanAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.takeTXKCReward(ctx, true)
}

func (h *Handler) OpenTXKCShop(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, _, err := h.loadActivityCharacter(ctx, "OpenTXKCShop")
	if err != nil {
		return nil, err
	}
	h.logger.Warn("OpenTXKCShop blocked without authoritative shop catalog", zap.Int64("character_id", characterID))
	h.sendActivityCallback(ctx, "onSystemSay", "Shop Khóa Học Đặc Huấn chưa có cấu hình vật phẩm chính xác")
	return nil, nil
}

func (h *Handler) mutateTXKCGold(ctx *rtmp.RPCContext, method string, mutation func(int64) (appspecialtraining.MutationResult, error)) (interface{}, error) {
	characterID, _, err := h.loadActivityCharacter(ctx, method)
	if err != nil {
		return nil, err
	}
	if h.specialTrainingSvc == nil {
		return nil, nil
	}
	result, err := mutation(characterID)
	if err != nil {
		h.handleTXKCError(ctx, method, characterID, err)
		return nil, nil
	}
	h.sendActivityCallback(ctx, "onAddMoney", float64(characterID), "gold", float64(-result.GoldSpent), float64(result.Gold))
	h.sendActivityCallback(ctx, "onUPP", map[string]interface{}{"gold": result.Gold})
	h.sendActivityCallback(ctx, "onTXKCGetData", result.Payload)
	return nil, nil
}

func (h *Handler) mutateTXKCState(ctx *rtmp.RPCContext, method string, mutation func(int64) (map[string]interface{}, error)) (interface{}, error) {
	characterID, _, err := h.loadActivityCharacter(ctx, method)
	if err != nil {
		return nil, err
	}
	if h.specialTrainingSvc == nil {
		return nil, nil
	}
	payload, err := mutation(characterID)
	if err != nil {
		h.handleTXKCError(ctx, method, characterID, err)
		return nil, nil
	}
	h.sendActivityCallback(ctx, "onTXKCGetData", payload)
	return nil, nil
}

func (h *Handler) takeTXKCReward(ctx *rtmp.RPCContext, premium bool) (interface{}, error) {
	characterID, _, err := h.loadActivityCharacter(ctx, "TakeTXKCReward")
	if err != nil {
		return nil, err
	}
	if h.specialTrainingSvc == nil || h.itemService == nil {
		return nil, nil
	}
	reward, level, payload, err := h.specialTrainingSvc.ReserveNextReward(ctx.Context, characterID, premium)
	if err != nil {
		h.handleTXKCError(ctx, "TakeTXKCReward", characterID, err)
		return nil, nil
	}
	added, err := h.itemService.AddTemplateItem(ctx.Context, characterID, appitem.TemplateRef{TableType: 29, TemplateID: reward.ItemID}, reward.Count, true, 0)
	if err != nil {
		if rollbackErr := h.specialTrainingSvc.RollbackReward(ctx.Context, characterID, premium, level); rollbackErr != nil {
			h.logger.Error("TXKC reward rollback failed", zap.Int64("character_id", characterID), zap.Error(rollbackErr))
		}
		h.handleTXKCError(ctx, "TakeTXKCReward", characterID, err)
		return nil, nil
	}
	h.sendActivityItemRewardCallbacks(ctx, added)
	h.sendActivityCallback(ctx, "onTXKCGetData", payload)
	return nil, nil
}

func (h *Handler) handleTXKCError(ctx *rtmp.RPCContext, method string, characterID int64, err error) {
	message := "Không thể thực hiện thao tác Khóa Học Đặc Huấn"
	switch {
	case errors.Is(err, appspecialtraining.ErrInactive):
		message = "Khóa Học Đặc Huấn hiện không trong thời gian mở"
	case errors.Is(err, appspecialtraining.ErrInsufficientGold):
		message = "Không đủ vàng"
	case errors.Is(err, appspecialtraining.ErrAlreadyPremium):
		message = "Đã mở khóa học cao cấp"
	case errors.Is(err, appspecialtraining.ErrAtMaxLevel):
		message = "Đã đạt cấp Khóa Học Đặc Huấn tối đa"
	case errors.Is(err, appspecialtraining.ErrTaskIncomplete):
		message = "Nhiệm vụ chưa hoàn thành"
	case errors.Is(err, appspecialtraining.ErrTaskClaimed), errors.Is(err, appspecialtraining.ErrRewardUnavailable):
		message = "Phần thưởng hiện không thể nhận"
	default:
		h.logger.Error(method+" failed", zap.Int64("character_id", characterID), zap.Error(err))
	}
	h.sendActivityCallback(ctx, "onSystemSay", message)
}
