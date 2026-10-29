// Open-sourced by BaoLT

// Gathering npc handlers manage resource queries and temporary reward flows.
package npc

import (
	"fmt"
	"strconv"
	"strings"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	farmhandler "mcgame-server/internal/presentation/rtmp/handlers/farm"
	pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) QueryFishPool(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.queryResourceNPC(args, "Dan ca"), nil
}

func (h *Handler) QueryHerb(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.queryResourceNPC(args, "Thao duoc"), nil
}

func (h *Handler) QueryGather(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.queryResourceNPC(args, "Tai nguyen"), nil
}

func (h *Handler) queryResourceNPC(args []interface{}, fallbackName string) map[string]interface{} {
	npcID, ok := farmhandler.FindFirstIntArg(args)
	if !ok || npcID == 0 {
		return map[string]interface{}{"name": fallbackName, "level": 1, "remain": 1}
	}

	payload := map[string]interface{}{"name": fallbackName, "level": 1, "remain": 1}
	if h.gameData == nil {
		return payload
	}

	npc := h.gameData.GetNPC(npcID)
	if npc == nil {
		return payload
	}
	if npc.Name != "" {
		payload["name"] = npc.Name
	}
	if level := int(npc.Lv); level > 0 {
		payload["level"] = level
	}
	if remain := int(npc.Num); remain > 0 {
		payload["remain"] = remain
	}

	return payload
}

func (h *Handler) tryHandleFishingNPCClick(ctx *rtmp.RPCContext, characterID int64, npcID int) (bool, error) {
	if h.gameData == nil || h.charService == nil {
		return false, nil
	}

	npc := h.gameData.GetNPC(npcID)
	if npc == nil || int(npc.Type) != 21 {
		return false, nil
	}

	return true, h.startFishing(ctx, characterID, npc)
}

func (h *Handler) tryHandleProductNPCClick(ctx *rtmp.RPCContext, characterID int64, npcID int) (bool, error) {
	if h.gameData == nil || h.charService == nil {
		return false, nil
	}

	npc := h.gameData.GetNPC(npcID)
	if npc == nil || !isProductNPC(npc) {
		return false, nil
	}

	payload := map[string]interface{}{"nid": npc.GetID(), "iid": strconv.Itoa(productRewardTemplateIDForNPC(npc))}
	return true, ctx.Connection.SendCallback("productInit", payload)
}

func (h *Handler) tryHandleHerbNPCClick(ctx *rtmp.RPCContext, characterID int64, npcID int) (bool, error) {
	if h.gameData == nil || h.charService == nil {
		return false, nil
	}

	npc := h.gameData.GetNPC(npcID)
	if npc == nil || int(npc.Type) != 23 {
		return false, nil
	}

	return true, h.startHerbGathering(ctx, characterID, npc)
}

func (h *Handler) OnProductReady(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if h.gameData == nil || h.charService == nil {
		return nil, pkgerrors.ErrSystemError
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	var npcID int
	var gloveID int
	for _, arg := range args {
		if npcID == 0 {
			if parsed, ok := farmhandler.FindIntInMap(arg, "npcId", "nid"); ok {
				npcID = parsed
			}
		}
		if gloveID == 0 {
			if parsed, ok := farmhandler.FindIntInMap(arg, "gloveId", "gid", "giid"); ok {
				gloveID = parsed
			}
		}
	}

	if npcID == 0 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	npc := h.gameData.GetNPC(npcID)
	if npc == nil || !isProductNPC(npc) {
		return "e", nil
	}
	if gloveID <= 0 {
		return "eg", nil
	}
	if h.itemService != nil {
		gloveItem, gloveErr := h.itemService.GetItemByID(ctx.Context, characterID, int64(gloveID))
		if gloveErr != nil || gloveItem == nil || gloveItem.SlotType != domainitem.SlotTypeEquipped {
			return "eg", nil
		}
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	if char == nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	const productMPCost = 30
	if char.CurrentMP < productMPCost {
		return "em", nil
	}

	char.CurrentMP -= productMPCost
	if err := h.charService.Update(ctx.Context, char); err != nil {
		return nil, err
	}

	if err := ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 13}); err != nil {
		return nil, err
	}

	_ = ctx.Connection.SendCallback("onUPP", map[string]interface{}{"currentMp": char.CurrentMP})
	if err := h.sendProductTempReward(ctx, characterID, npc); err != nil {
		return nil, err
	}

	remaining := productRemainingForNPC(npc)
	_ = ctx.Connection.SendCallback("onProduct", remaining)
	return remaining, nil
}

func (h *Handler) OnProductCancel(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) sendProductTempReward(ctx *rtmp.RPCContext, characterID int64, npc *models.NpcTemplate) error {
	rewardTemplateID := productRewardTemplateIDForNPC(npc)
	rewardTemplate := h.getItemTemplate(rewardTemplateID)
	rewardName := h.getItemTemplateName(rewardTemplateID)

	slotIndex := 1
	curNum := 1
	slotData := map[string]interface{}{"c": productBindValue(rewardTemplate), "b": productBindValue(rewardTemplate), "t": rewardTemplateID, "n": 1}

	if h.itemService != nil {
		addedItem, err := h.itemService.AddItemToSlot(ctx.Context, characterID, rewardTemplateID, domainitem.ItemTypeMaterial, 1, productBindValue(rewardTemplate) > 0, domainitem.SlotTypeTempBag)
		if err != nil {
			return err
		}
		if addedItem != nil {
			slotIndex = addedItem.SlotIndex + 1
			slotData = map[string]interface{}{"c": productBindValue(rewardTemplate), "b": productBindValue(rewardTemplate), "t": addedItem.TemplateID, "n": addedItem.StackCount}
			curNum = addedItem.StackCount
		}
	}

	effectData := map[string]interface{}{"s": 1, "tt": "6", "i": rewardTemplateID, "c": productBindValue(rewardTemplate), "t": 29, "n": rewardName, "q": productQualityValue(rewardTemplate)}
	return ctx.Connection.SendCallback("onAddTempDirect", effectData, slotIndex, slotData, curNum)
}

func (h *Handler) countTempBagItems(ctx *rtmp.RPCContext, characterID int64) int {
	if h.itemService == nil {
		return 1
	}
	items, err := h.itemService.GetAllItems(ctx.Context, characterID)
	if err != nil {
		return 1
	}
	count := 0
	for _, item := range items {
		if item != nil && item.SlotType == domainitem.SlotTypeTempBag {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func (h *Handler) getItemTemplateName(templateID int) string {
	if itemTemplate := h.getItemTemplate(templateID); itemTemplate != nil && itemTemplate.Name != "" {
		return itemTemplate.Name
	}
	return fmt.Sprintf("Vat pham %d", templateID)
}

func (h *Handler) getItemTemplate(templateID int) *models.ItemTemplateTemplate {
	if h.gameData != nil {
		if item := h.gameData.GetItem(templateID); item != nil {
			return item
		}
	}
	return nil
}

func productRewardTemplateIDForNPC(npc *models.NpcTemplate) int {
	if rewardTemplateID, ok := extractPositiveTemplateID(npc); ok {
		return rewardTemplateID
	}
	return 18
}

func productRemainingForNPC(npc *models.NpcTemplate) int {
	if npc == nil {
		return 1
	}
	remain := int(npc.Num) - 1
	if remain <= 0 {
		return 1
	}
	return remain
}

func productBindValue(itemTemplate *models.ItemTemplateTemplate) int {
	if itemTemplate == nil {
		return 1
	}
	if itemTemplate.BindType > 0 {
		return int(itemTemplate.BindType)
	}
	return 1
}

func productQualityValue(itemTemplate *models.ItemTemplateTemplate) int {
	if itemTemplate == nil {
		return 5
	}
	if itemTemplate.Color > 0 {
		return int(itemTemplate.Color)
	}
	if itemTemplate.ColorCode > 0 {
		return int(itemTemplate.ColorCode)
	}
	return 5
}

func isProductNPC(npc *models.NpcTemplate) bool {
	if npc == nil {
		return false
	}
	return int(npc.Type) == 24
}

func extractPositiveTemplateID(npc *models.NpcTemplate) (int, bool) {
	if npc == nil {
		return 0, false
	}
	for _, raw := range []string{npc.SubType, npc.Item} {
		if raw == "" {
			continue
		}
		for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == '|' || r == ',' || r == ';' || r == ':' || r == ' '
		}) {
			prefix := leadingPositiveDigits(token)
			if prefix == "" {
				continue
			}
			if parsed, err := strconv.Atoi(prefix); err == nil && parsed > 0 {
				return parsed, true
			}
		}
	}
	return 0, false
}

func leadingPositiveDigits(value string) string {
	if value == "" {
		return ""
	}
	if value[0] < '0' || value[0] > '9' {
		return ""
	}
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	return value[:end]
}
