// Open-sourced by BaoLT

// Achievement reward grants fail closed and return errors to claim orchestration.
package achievement

import (
	"fmt"
	"strconv"

	appachievement "mcgame-server/internal/application/achievement"
	domainachievement "mcgame-server/internal/domain/achievement"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
)

func (h *Handler) grantReward(ctx *rtmp.RPCContext, characterID, achievementID int64) (domainachievement.RewardReceipt, error) {
	receipt := domainachievement.RewardReceipt{}
	reward, ok := appachievement.RewardForAchievement(achievementID)
	if !ok {
		return receipt, appachievement.ErrAchievementRewardUnknown
	}
	if reward.TemplateID <= 0 || reward.Quantity <= 0 {
		return receipt, fmt.Errorf("invalid achievement reward definition")
	}
	grantKey := "achievement:" + strconv.FormatInt(characterID, 10) + ":" + strconv.FormatInt(achievementID, 10)

	switch reward.Kind {
	case appachievement.RewardKindItem:
		return h.grantItem(ctx, characterID, reward.TemplateID, reward.Quantity, grantKey)
	case appachievement.RewardKindTitle:
		return h.grantTitle(ctx, characterID, reward.TemplateID)
	case appachievement.RewardKindMedal:
		return h.grantMedal(ctx, characterID, reward.TemplateID, reward.Quantity, grantKey)
	default:
		return receipt, fmt.Errorf("unsupported achievement reward kind")
	}
}

func (h *Handler) grantItem(ctx *rtmp.RPCContext, characterID int64, templateID, quantity int, grantKey string) (domainachievement.RewardReceipt, error) {
	receipt := domainachievement.RewardReceipt{}
	if h.itemService == nil {
		return receipt, fmt.Errorf("achievement item service not configured")
	}
	if h.gameDataManager == nil || h.gameDataManager.GetItem(templateID) == nil {
		return receipt, fmt.Errorf("achievement item %d not found", templateID)
	}
	added, err := h.itemService.AddItemToSlotIdempotent(ctx.Context, characterID, templateID, domainitem.ItemTypeConsumable, quantity, false, domainitem.SlotTypeBag, grantKey)
	if err != nil {
		return receipt, err
	}
	if added == nil {
		return receipt, fmt.Errorf("achievement item grant returned no item")
	}
	receipt.ItemID = added.ID
	return receipt, nil
}

func (h *Handler) grantTitle(ctx *rtmp.RPCContext, characterID int64, titleID int) (domainachievement.RewardReceipt, error) {
	receipt := domainachievement.RewardReceipt{}
	if h.titleService == nil {
		return receipt, fmt.Errorf("achievement title service not configured")
	}
	if h.gameDataManager == nil || h.gameDataManager.GetTitle(titleID) == nil {
		return receipt, fmt.Errorf("achievement title %d not found", titleID)
	}
	_, err := h.titleService.GrantTitle(ctx.Context, characterID, titleID)
	if err != nil {
		return receipt, err
	}
	receipt.TitleID = titleID
	return receipt, nil
}

func (h *Handler) grantMedal(ctx *rtmp.RPCContext, characterID int64, templateID, quantity int, grantKey string) (domainachievement.RewardReceipt, error) {
	receipt := domainachievement.RewardReceipt{}
	if h.medalService == nil {
		return receipt, fmt.Errorf("achievement medal service not configured")
	}
	if h.gameDataManager == nil || h.gameDataManager.GetMedal(templateID) == nil {
		return receipt, fmt.Errorf("achievement medal %d not found", templateID)
	}
	slot, count, err := h.medalService.AddMedalToBagIdempotent(ctx.Context, characterID, templateID, quantity, grantKey)
	if err != nil {
		return receipt, err
	}
	receipt.MedalSlot = slot
	receipt.MedalCount = count
	return receipt, nil
}

func (h *Handler) sendRewardCallbacks(ctx *rtmp.RPCContext, characterID, achievementID int64, receipt domainachievement.RewardReceipt) error {
	reward, ok := appachievement.RewardForAchievement(achievementID)
	if !ok || ctx.Connection == nil {
		return nil
	}
	switch reward.Kind {
	case appachievement.RewardKindItem:
		if h.itemService == nil || h.gameDataManager == nil || receipt.ItemID <= 0 {
			return fmt.Errorf("invalid achievement item receipt")
		}
		template := h.gameDataManager.GetItem(reward.TemplateID)
		if template == nil {
			return fmt.Errorf("achievement item %d not found", reward.TemplateID)
		}
		added, err := h.itemService.GetItemByID(ctx.Context, characterID, receipt.ItemID)
		if err != nil {
			return err
		}
		if added.TemplateID != reward.TemplateID {
			return fmt.Errorf("achievement item receipt template mismatch")
		}
		_ = ctx.Connection.SendCallback("onAddItem", map[string]interface{}{"i": reward.TemplateID, "q": reward.Quantity, "s": "", "c": added.ColorCode, "t": 28, "n": template.Name, "tt": int(template.Kind)})
		_ = ctx.Connection.SendCallback("onAddCharactorSlot", added.ToDTO())
	case appachievement.RewardKindTitle:
		if h.titleService == nil || h.gameDataManager == nil || receipt.TitleID != reward.TemplateID {
			return fmt.Errorf("invalid achievement title receipt")
		}
		template := h.gameDataManager.GetTitle(receipt.TitleID)
		if template == nil {
			return fmt.Errorf("achievement title %d not found", receipt.TitleID)
		}
		result, err := h.titleService.GrantTitle(ctx.Context, characterID, receipt.TitleID)
		if err != nil {
			return err
		}
		rtmputils.SendTitleGrantedCallback(ctx.Connection, receipt.TitleID, result.Titles, result.SpecialTitles, result.IsSpecial)
		_ = ctx.Connection.SendCallback("onAchieveTitle", template.N)
	case appachievement.RewardKindMedal:
		if receipt.MedalSlot <= 0 || receipt.MedalCount <= 0 {
			return fmt.Errorf("invalid achievement medal receipt")
		}
		_ = ctx.Connection.SendCallback("updateMedalInBag", map[string]interface{}{
			"i": receipt.MedalSlot,
			"t": reward.TemplateID,
			"n": receipt.MedalCount,
		})
	}
	return nil
}
