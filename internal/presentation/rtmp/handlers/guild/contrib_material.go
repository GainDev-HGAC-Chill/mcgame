// Open-sourced by BaoLT

package guild

import (
	"math"
	"strconv"

	appguild "mcgame-server/internal/application/guild"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	itemKindMaterial             = 6
	guildMaterialNormalTemplate  = 2026
	guildMaterialRareTemplate    = 2027
	guildMaterialSpecialTemplate = 2039
	contribTypeDonate            = 1
	guildWarehouseSIDBase        = 501
)

func (h *Handler) ContribMaterial(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	request, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}

	slotID, err := parseInt64Arg(request["slotId"])
	if err != nil {
		return nil, err
	}
	countValue, err := parseIntArg(request["num"])
	if err != nil {
		return nil, err
	}
	if countValue <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}

	if h.itemService == nil || h.guildService == nil {
		return nil, pkgerrors.ErrSystemError
	}

	itemData, err := h.itemService.GetItemByID(ctx.Context, characterID, slotID)
	if err != nil {
		return nil, err
	}
	if itemData == nil {
		return nil, pkgerrors.ErrNotFound
	}
	if itemData.SlotType != domainitem.SlotTypeBag || itemData.StackCount < countValue {
		return nil, pkgerrors.ErrInvalidInput
	}

	template := h.itemService.GetItemTemplate(itemData.TemplateID)
	if template == nil {
		return nil, pkgerrors.ErrNotFound
	}
	materialTemplateID, guildMaterialCount, personalContrib, err := guildMaterialContribution(template, itemData.ColorCode, countValue)
	if err != nil {
		return nil, err
	}
	warehouseTemplate := h.itemService.GetItemTemplate(materialTemplateID)
	if warehouseTemplate == nil || warehouseTemplate.StackMax <= 0 || warehouseTemplate.StackMax > math.MaxInt32 || math.Trunc(warehouseTemplate.StackMax) != warehouseTemplate.StackMax {
		return nil, pkgerrors.ErrInvalidInput
	}

	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}
	if err := h.requireGuildBuildingFunction(ctx, guildItem.ID, "showDonatePanel"); err != nil {
		return nil, err
	}
	committed, err := h.guildService.ContributeMaterialAtomic(ctx.Context, appguild.ContributeMaterialCommand{
		GuildID: guildItem.ID, CharacterID: characterID, ItemID: slotID, ItemTemplateID: itemData.TemplateID,
		ItemCount: countValue, WarehouseTemplateID: materialTemplateID, WarehouseCount: guildMaterialCount,
		WarehouseMaxStack: int(warehouseTemplate.StackMax), WarehousePageCount: guildItem.BankPageCount, DonateGain: int64(personalContrib),
	})
	if err != nil {
		return nil, guildError(err)
	}

	itemData.StackCount = committed.ItemRemaining
	if committed.ItemRemaining <= 0 {
		_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(itemData.ID), itemData.CalculateSID())
	} else {
		_ = ctx.Connection.SendCallback("onAddCharactorSlot", itemData.ToDTO())
	}

	h.sendContributionState(characterID, "")

	return 1, nil
}

func (h *Handler) DropGuildItem(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	slotID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	if h.guildService == nil {
		return nil, pkgerrors.ErrSystemError
	}

	guildItem, selfMember, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil || selfMember == nil {
		return nil, pkgerrors.ErrNotFound
	}
	if err := h.requireGuildBuildingFunction(ctx, guildItem.ID, "showGuildWarehouse"); err != nil {
		return nil, err
	}
	committed, err := h.guildService.DropGuildWarehouseAtomic(ctx.Context, appguild.WarehouseDropCommand{GuildID: guildItem.ID, ActorID: characterID, SlotID: slotID})
	if err != nil {
		return nil, guildError(err)
	}

	h.broadcastGuild(guildItem, "onDelGuildSlot", float64(slotID))
	h.broadcastGuild(guildItem, "onDelGuildItem", map[string]interface{}{
		"tid": committed.TemplateID,
		"num": committed.StackCount,
	})

	return map[string]interface{}{"ok": true}, nil
}

func (h *Handler) MoveGuildItem(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	sourceSID, err := parseIntArg(args[0])
	if err != nil {
		return nil, err
	}
	targetSID, err := parseIntArg(args[1])
	if err != nil {
		return nil, err
	}
	guildItem, selfMember, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil || selfMember == nil {
		return nil, pkgerrors.ErrNotFound
	}
	if err := h.requireGuildBuildingFunction(ctx, guildItem.ID, "showGuildWarehouse"); err != nil {
		return nil, err
	}
	sourceSlot, ok := guildWarehouseSlotBySID(guildItem.Warehouse, sourceSID)
	if !ok {
		return nil, pkgerrors.ErrNotFound
	}
	if h.itemService == nil {
		return nil, pkgerrors.ErrSystemError
	}
	template := h.itemService.GetItemTemplate(sourceSlot.TemplateID)
	if template == nil || template.StackMax <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	targetIndex := guildWarehouseSlotIndexFromSID(targetSID)
	result, err := h.guildService.MoveGuildWarehouseAtomic(ctx.Context, appguild.WarehouseMoveCommand{
		GuildID: guildItem.ID, ActorID: characterID, SourceID: sourceSlot.ID,
		SourceIndex: guildWarehouseSlotIndexFromSID(sourceSID), TargetIndex: targetIndex,
		TemplateID: sourceSlot.TemplateID, MaxStack: int(template.StackMax), PageCount: guildItem.BankPageCount,
	})
	if err != nil {
		return nil, guildError(err)
	}
	switch result.Mode {
	case "move":
		h.broadcastGuild(guildItem, "onMoveGuildItem", map[string]interface{}{
			"sSid": sourceSID,
			"tSid": targetSID,
			"sId":  result.SourceID,
		})
	case "merge":
		h.broadcastGuild(guildItem, "onStackGuildItemInto", map[string]interface{}{
			"sSid":      sourceSID,
			"tSid":      targetSID,
			"sId":       result.SourceID,
			"tId":       result.TargetID,
			"tStackNum": result.TargetCount,
		})
	case "partial":
		h.broadcastGuild(guildItem, "onStackGuildItem", map[string]interface{}{
			"sSid":      sourceSID,
			"tSid":      targetSID,
			"sId":       result.SourceID,
			"tId":       result.TargetID,
			"tStackNum": result.TargetCount,
			"sStackNum": result.SourceCount,
		})
	default:
		return nil, pkgerrors.ErrSystemError
	}
	return map[string]interface{}{"ok": true}, nil
}

func guildMaterialContribution(template *models.ItemTemplateTemplate, color int, count int) (int, int, int, error) {
	templateID := int(template.ID)
	if templateID != guildMaterialSpecialTemplate {
		if int(template.Kind) != itemKindMaterial || color < 2 {
			return 0, 0, 0, pkgerrors.ErrInvalidInput
		}
	}

	factor := math.Pow(float64(color), 5)
	switch materialBucketTemplate(templateID) {
	case guildMaterialNormalTemplate:
		guildCount := int(math.Floor((float64(count) * factor) / 8))
		contrib := int(math.Floor(((float64(count) * factor) * 50) / 8))
		return guildMaterialNormalTemplate, guildCount, contrib, nil
	case guildMaterialRareTemplate:
		guildCount := int(math.Floor((float64(count) * factor) / 8))
		contrib := int(math.Floor(((float64(count) * factor) * 100) / 8))
		return guildMaterialRareTemplate, guildCount, contrib, nil
	case guildMaterialSpecialTemplate:
		return guildMaterialSpecialTemplate, count, count * 100, nil
	default:
		return 0, 0, 0, pkgerrors.ErrInvalidInput
	}
}

func materialBucketTemplate(templateID int) int {
	switch templateID {
	case guildMaterialSpecialTemplate:
		return guildMaterialSpecialTemplate
	case 24, 21, 23, 26, 28:
		return guildMaterialRareTemplate
	case 16, 7, 18, 22, 25, 27:
		return guildMaterialNormalTemplate
	default:
		return 0
	}
}

type guildWarehouseSlotData struct {
	ID         int64
	SID        int
	TemplateID int
	StackCount int
}

func guildWarehouseSlotDTO(warehouse map[string]interface{}, slotID int64) (guildWarehouseSlotData, bool) {
	raw, ok := warehouse[strconv.FormatInt(slotID, 10)]
	if !ok {
		return guildWarehouseSlotData{}, false
	}
	slot, ok := raw.(map[string]interface{})
	if !ok {
		return guildWarehouseSlotData{}, false
	}
	return guildWarehouseSlotData{
		ID:         slotID,
		SID:        safeInt(slot["sid"]),
		TemplateID: safeInt(slot["tid"]),
		StackCount: safeInt(slot["stackNum"]),
	}, true
}

func guildWarehouseSlotBySID(warehouse map[string]interface{}, sid int) (guildWarehouseSlotData, bool) {
	for key, raw := range warehouse {
		slot, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if safeInt(slot["sid"]) != sid {
			continue
		}
		slotID, _ := strconv.ParseInt(key, 10, 64)
		return guildWarehouseSlotData{
			ID:         slotID,
			SID:        sid,
			TemplateID: safeInt(slot["tid"]),
			StackCount: safeInt(slot["stackNum"]),
		}, true
	}
	return guildWarehouseSlotData{}, false
}

func guildWarehouseSlotIndexFromSID(sid int) int {
	if sid < guildWarehouseSIDBase {
		return -1
	}
	return sid - guildWarehouseSIDBase
}

func guildWarehouseStackMax(template *models.ItemTemplateTemplate) int {
	if template == nil || int(template.StackMax) <= 0 {
		return 99999
	}
	return int(template.StackMax)
}

func safeInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		n, _ := strconv.Atoi(typed)
		return n
	default:
		return 0
	}
}
