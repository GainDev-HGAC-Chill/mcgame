// Open-sourced by BaoLT

// Achievement reward grants fail closed and return errors to claim orchestration.
package achievement

import (
	"fmt"

	appachievement "mcgame-server/internal/application/achievement"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
)

func (h *Handler) grantReward(ctx *rtmp.RPCContext, characterID, achievementID int64) error {
	reward, ok := appachievement.RewardForAchievement(achievementID)
	if !ok {
		return appachievement.ErrAchievementRewardUnknown
	}
	if reward.TemplateID <= 0 || reward.Quantity <= 0 {
		return fmt.Errorf("invalid achievement reward definition")
	}

	switch reward.Kind {
	case appachievement.RewardKindItem:
		return h.grantItem(ctx, characterID, reward.TemplateID, reward.Quantity)
	case appachievement.RewardKindTitle:
		return h.grantTitle(ctx, characterID, reward.TemplateID)
	case appachievement.RewardKindMedal:
		return h.grantMedal(ctx, characterID, reward.TemplateID, reward.Quantity)
	default:
		return fmt.Errorf("unsupported achievement reward kind")
	}
}

func (h *Handler) grantItem(ctx *rtmp.RPCContext, characterID int64, templateID, quantity int) error {
	if h.itemService == nil {
		return fmt.Errorf("achievement item service not configured")
	}
	if h.gameDataManager == nil || h.gameDataManager.GetItem(templateID) == nil {
		return fmt.Errorf("achievement item %d not found", templateID)
	}
	added, err := h.itemService.AddItemToSlot(ctx.Context, characterID, templateID, domainitem.ItemTypeConsumable, quantity, false, domainitem.SlotTypeBag)
	if err != nil {
		return err
	}
	if added == nil {
		return fmt.Errorf("achievement item grant returned no item")
	}
	if ctx.Connection != nil {
		template := h.gameDataManager.GetItem(templateID)
		_ = ctx.Connection.SendCallback("onAddItem", map[string]interface{}{
			"i":  templateID,
			"q":  quantity,
			"s":  "",
			"c":  added.ColorCode,
			"t":  28,
			"n":  template.Name,
			"tt": int(template.Kind),
		})
		_ = ctx.Connection.SendCallback("onAddCharactorSlot", added.ToDTO())
	}
	return nil
}

func (h *Handler) grantTitle(ctx *rtmp.RPCContext, characterID int64, titleID int) error {
	if h.titleService == nil {
		return fmt.Errorf("achievement title service not configured")
	}
	if h.gameDataManager == nil || h.gameDataManager.GetTitle(titleID) == nil {
		return fmt.Errorf("achievement title %d not found", titleID)
	}
	result, err := h.titleService.GrantTitle(ctx.Context, characterID, titleID)
	if err != nil {
		return err
	}
	if result.Added && ctx.Connection != nil {
		template := h.gameDataManager.GetTitle(titleID)
		rtmputils.SendTitleGrantedCallback(ctx.Connection, titleID, result.Titles, result.SpecialTitles, result.IsSpecial)
		_ = ctx.Connection.SendCallback("onAchieveTitle", template.N)
	}
	return nil
}

func (h *Handler) grantMedal(ctx *rtmp.RPCContext, characterID int64, templateID, quantity int) error {
	if h.medalService == nil {
		return fmt.Errorf("achievement medal service not configured")
	}
	if h.gameDataManager == nil || h.gameDataManager.GetMedal(templateID) == nil {
		return fmt.Errorf("achievement medal %d not found", templateID)
	}
	slot, count, err := h.medalService.AddMedalToBag(ctx.Context, characterID, templateID, quantity)
	if err != nil {
		return err
	}
	if ctx.Connection != nil {
		_ = ctx.Connection.SendCallback("updateMedalInBag", map[string]interface{}{
			"i": slot,
			"t": templateID,
			"n": count,
		})
	}
	return nil
}
