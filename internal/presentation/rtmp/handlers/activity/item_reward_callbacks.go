package activity

import (
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) sendActivityItemRewardCallbacks(ctx *rtmp.RPCContext, added *domainitem.Item) {
	if h == nil || ctx == nil || ctx.Connection == nil || added == nil {
		return
	}
	notice, slot := h.activityItemRewardPayloads(added)
	_ = ctx.Connection.SendCallback("onAddItem", notice)
	_ = ctx.Connection.SendCallback("onAddCharactorSlot", slot)
}

func (h *Handler) activityItemRewardPayloads(added *domainitem.Item) (map[string]interface{}, map[string]interface{}) {
	slot := added.ToDTO()
	if h != nil && h.itemService != nil {
		slot = h.itemService.BuildClientItemDTO(added)
	}
	name := ""
	kind := 0
	tableType := 29
	if added.ItemType == domainitem.ItemTypeEquipment {
		tableType = 19
	}
	if h != nil && h.gameData != nil {
		if tableType == 19 {
			if tpl := h.gameData.GetEquipment(added.TemplateID); tpl != nil {
				name = tpl.Name
				kind = int(tpl.Kind)
			}
		} else if tpl := h.gameData.GetItem(added.TemplateID); tpl != nil {
			name = tpl.Name
			kind = int(tpl.Kind)
		}
	}
	notice := map[string]interface{}{
		"i":  added.TemplateID,
		"q":  added.StackCount,
		"s":  "",
		"c":  added.ColorCode,
		"t":  tableType,
		"n":  name,
		"tt": kind,
	}
	return notice, slot
}
