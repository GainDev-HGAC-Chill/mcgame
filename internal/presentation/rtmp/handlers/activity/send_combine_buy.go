// Open-sourced by BaoLT

// Send Combine purchase handler: validates selection, deducts currency, grants items, and tracks buy counts.
package activity

import (
	"errors"
	"strconv"
	"strings"

	"go.uber.org/zap"
	appactivity "mcgame-server/internal/application/activity"
	appitem "mcgame-server/internal/application/item"
	domainactivity "mcgame-server/internal/domain/activity"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	sendCombineMsgLimitReached      = "Reached purchase limit for this package"
	sendCombineMsgNotEnoughCurrency = "Not enough currency"
	sendCombineMsgInventoryIssue    = "Bag is full or reward cannot be added"
	sendCombineMsgInvalidData       = "Invalid send combine configuration"
)

func (h *Handler) BuySendCombine(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, characterID, err := h.characterID(ctx, "BuySendCombine")
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	if h.itemService == nil || h.sendCombineService == nil {
		return nil, pkgerrors.ErrSystemError
	}

	activityID, ok := parseSendCombineActivityID(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}

	activity, exists := getSendCombineActivityByID(activityID)
	if !exists {
		return nil, pkgerrors.ErrInvalidInput
	}

	buyCounts, err := h.sendCombineService.BuyCounts(ctx.Context, characterID, activity.End)
	if err != nil {
		return nil, err
	}
	currentBuyNum := buyCounts[activityID]
	if currentBuyNum >= activity.Limit {
		h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgLimitReached)
		return currentBuyNum, nil
	}

	selectedAwards := selectSendCombineAwards(activity, parseSendCombineSelectFlags(args))
	if len(selectedAwards) == 0 {
		h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgInvalidData)
		return currentBuyNum, nil
	}

	if !strings.EqualFold(strings.TrimSpace(activity.Currency), "gold") {
		h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgInvalidData)
		return currentBuyNum, nil
	}
	totalCost := sendCombineDiscountTotal(selectedAwards)
	commandAwards := make([]domainactivity.SendCombineAward, 0, len(selectedAwards))
	for _, award := range selectedAwards {
		templateID, parseErr := strconv.Atoi(award.ItemID)
		if parseErr != nil || templateID <= 0 {
			h.logger.Warn("BuySendCombine: invalid award template id",
				zap.Uint32("conn_id", ctx.ConnID),
				zap.Int64("character_id", characterID),
				zap.String("activity_id", activityID),
				zap.String("item_id", award.ItemID))
			h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgInvalidData)
			return currentBuyNum, nil
		}

		ref := appitem.TemplateRef{
			TableType:  award.ItemType,
			TemplateID: templateID,
		}

		itemType, resolveErr := h.itemService.ResolveTemplateType(ref)
		if resolveErr != nil {
			h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgInvalidData)
			return currentBuyNum, nil
		}
		commandAwards = append(commandAwards, domainactivity.SendCombineAward{TemplateID: templateID, ItemType: itemType, Count: award.StackNum, IsBound: award.Binded == 1})
	}
	result, err := h.sendCombineService.Purchase(ctx.Context, domainactivity.SendCombinePurchaseCommand{
		CharacterID: characterID,
		ActivityID:  activityID,
		EventEndMs:  activity.End,
		Limit:       activity.Limit,
		Cost:        totalCost,
		Awards:      commandAwards,
	})
	if err != nil {
		switch {
		case errors.Is(err, appactivity.ErrSendCombineLimitReached):
			h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgLimitReached)
		case errors.Is(err, pkgerrors.ErrInsufficientFunds):
			h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgNotEnoughCurrency)
		case errors.Is(err, pkgerrors.ErrInventoryFull):
			h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgInventoryIssue)
		case errors.Is(err, appactivity.ErrSendCombineExpired), errors.Is(err, pkgerrors.ErrInvalidInput):
			h.sendSendCombineNotice(ctx, "onRedMsg", sendCombineMsgInvalidData)
		default:
			return nil, err
		}
		return currentBuyNum, nil
	}

	if ctx.Connection != nil {
		for _, added := range result.Items {
			h.sendActivityItemRewardCallbacks(ctx, added)
		}
		_ = ctx.Connection.SendCallback("onAddMoney", float64(characterID), "gold", float64(-totalCost), float64(result.Gold))
		_ = ctx.Connection.SendCallback("onUPP", map[string]interface{}{"gold": result.Gold})
		rtmputils.CheckBagCapacityWarning(ctx.Context, ctx.Connection, h.itemService, characterID)
	}

	h.logger.Info("BuySendCombine: purchase completed",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.Int64("character_id", characterID),
		zap.String("activity_id", activityID),
		zap.Int("buy_num", result.BuyCount),
		zap.Int64("total_cost", totalCost),
		zap.String("currency", activity.Currency),
		zap.Int("award_count", len(selectedAwards)))

	return result.BuyCount, nil
}

func parseSendCombineActivityID(value interface{}) (string, bool) {
	switch raw := value.(type) {
	case string:
		if raw == "" {
			return "", false
		}
		return raw, true
	case float64:
		return strconv.Itoa(int(raw)), true
	case int:
		return strconv.Itoa(raw), true
	case int64:
		return strconv.Itoa(int(raw)), true
	default:
		return "", false
	}
}

func parseSendCombineSelectFlags(args []interface{}) map[int]bool {
	flags := map[int]bool{
		1: false,
		2: false,
		3: false,
	}

	if len(args) > 1 {
		flags[1] = parseBoolArg(args[1])
	}
	if len(args) > 2 {
		flags[2] = parseBoolArg(args[2])
	}
	if len(args) > 3 {
		flags[3] = parseBoolArg(args[3])
	}

	return flags
}

func parseBoolArg(value interface{}) bool {
	switch raw := value.(type) {
	case bool:
		return raw
	case string:
		normalized := strings.ToLower(strings.TrimSpace(raw))
		return normalized == "true" || normalized == "1" || normalized == "yes"
	case float64:
		return raw != 0
	case int:
		return raw != 0
	case int64:
		return raw != 0
	default:
		return false
	}
}

func selectSendCombineAwards(activity sendCombineActivity, flags map[int]bool) []sendCombineAward {
	selected := make([]sendCombineAward, 0, len(activity.Award))
	for index, award := range activity.Award {
		if index == 0 {
			selected = append(selected, award)
			continue
		}

		if strings.EqualFold(award.Checkable, "false") {
			if flags[index] {
				selected = append(selected, award)
			}
			continue
		}

		selected = append(selected, award)
	}

	return selected
}

func sendCombineDiscountTotal(awards []sendCombineAward) int64 {
	var total int64
	for _, award := range awards {
		total += int64(award.Discount)
	}
	return total
}

func (h *Handler) sendSendCombineNotice(ctx *rtmp.RPCContext, callbackName string, message string) {
	if ctx.Connection == nil {
		return
	}
	_ = ctx.Connection.SendCallback(callbackName, message)
}
