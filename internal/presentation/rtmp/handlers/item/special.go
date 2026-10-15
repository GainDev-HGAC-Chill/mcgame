// Open-sourced by BaoLT

// Special item RPC handlers: rename card, sex change card, good card, wedding/red bags, intimacy items.
package item

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

const (
	renameCardTemplateID    = 2126
	sexChangeCardTemplateID = 2844
	goodCardTemplateID      = 122
	largeRoseTemplateID     = 32
	chocolateTemplateID     = 1302
	roseTemplateID          = 1303
	chocolateBoxTemplateID  = 1315
)

func (h *Handler) ChangeNameByCard(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	newName, ok := args[0].(string)
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	newName = strings.TrimSpace(newName)

	nameLen := utf8.RuneCountInString(newName)
	if nameLen < 2 || nameLen > 12 {
		ctx.Connection.SendCallback("onSystemSay", "Tên phải từ 2 đến 12 ký tự.")
		return nil, nil
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	consumed, err := h.itemService.ConsumeItemsByTemplateID(ctx.Context, characterID, renameCardTemplateID, 1)
	if err != nil {
		h.logger.Error("ChangeNameByCard: failed to consume card", zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}
	if !consumed {
		ctx.Connection.SendCallback("onSystemSay", "Bạn không có Thẻ Đổi Tên.")
		return nil, nil
	}

	char.Name = newName
	if err := h.charService.Update(ctx.Context, char); err != nil {
		h.logger.Error("ChangeNameByCard: failed to update character", zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}

	ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"name": newName,
	})

	h.logger.Info("ChangeNameByCard: name changed",
		zap.Int64("character_id", characterID),
		zap.String("new_name", newName))

	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) ChangeCharSex(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	consumed, err := h.itemService.ConsumeItemsByTemplateID(ctx.Context, characterID, sexChangeCardTemplateID, 1)
	if err != nil {
		h.logger.Error("ChangeCharSex: failed to consume card", zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}
	if !consumed {
		ctx.Connection.SendCallback("onSystemSay", "Bạn không có Thẻ Đổi Giới Tính.")
		return nil, nil
	}

	if char.Gender == 0 {
		char.Gender = 1
	} else {
		char.Gender = 0
	}

	if err := h.charService.Update(ctx.Context, char); err != nil {
		h.logger.Error("ChangeCharSex: failed to update character", zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}

	ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"gender": char.Gender,
	})

	h.logger.Info("ChangeCharSex: gender changed",
		zap.Int64("character_id", characterID),
		zap.Int("new_gender", char.Gender))

	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) UseGoodCard(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	consumed, err := h.itemService.ConsumeItemsByTemplateID(ctx.Context, characterID, goodCardTemplateID, 1)
	if err != nil {
		h.logger.Error("UseGoodCard: failed to consume card", zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}
	if !consumed {
		ctx.Connection.SendCallback("onSystemSay", "Bạn không có Thẻ May Mắn.")
		return nil, nil
	}

	ctx.Connection.SendCallback("onSystemSay", "Đã sử dụng Thẻ May Mắn.")

	h.logger.Info("UseGoodCard: card used",
		zap.Int64("character_id", characterID))

	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) UseWeddingBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	slotID, ok := parseFlexibleInt64(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	it, err := h.itemService.GetItemByID(ctx.Context, characterID, slotID)
	if err != nil {
		return rtmp.ErrorToResponse(pkgerrors.ErrItemNotFound), nil
	}

	oldSID := it.CalculateSID()
	usedItem, result, err := h.itemService.UseItem(ctx.Context, char, it.ID, nil)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	if err := h.charService.Update(ctx.Context, char); err != nil {
		h.logger.Error("UseWeddingBag: failed to update character", zap.Error(err))
	}

	if usedItem.StackCount <= 0 {
		ctx.Connection.SendCallback("onDelCharactorSlot", float64(usedItem.ID), float64(oldSID))
	} else {
		ctx.Connection.SendCallback("onAddCharactorSlot", h.itemDTO(usedItem))
	}

	h.sendEffectCallbacks(ctx, result, char)

	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) UseRedBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	slotID, ok := parseFlexibleInt64(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	it, err := h.itemService.GetItemByID(ctx.Context, characterID, slotID)
	if err != nil {
		return rtmp.ErrorToResponse(pkgerrors.ErrItemNotFound), nil
	}

	oldSID := it.CalculateSID()
	usedItem, result, err := h.itemService.UseItem(ctx.Context, char, it.ID, nil)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	if err := h.charService.Update(ctx.Context, char); err != nil {
		h.logger.Error("UseRedBag: failed to update character", zap.Error(err))
	}

	if usedItem.StackCount <= 0 {
		ctx.Connection.SendCallback("onDelCharactorSlot", float64(usedItem.ID), float64(oldSID))
	} else {
		ctx.Connection.SendCallback("onAddCharactorSlot", h.itemDTO(usedItem))
	}

	h.sendEffectCallbacks(ctx, result, char)

	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) UseIntimacyItem(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	targetName, ok := args[0].(string)
	if !ok || strings.TrimSpace(targetName) == "" {
		return nil, pkgerrors.ErrInvalidArgs
	}

	quantityRaw, ok := parseFlexibleInt64(args[1])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	if quantityRaw <= 0 || quantityRaw > int64(^uint(0)>>1) {
		return nil, pkgerrors.ErrInvalidArgs
	}
	quantity := int(quantityRaw)

	itemIDRaw, ok := parseFlexibleInt64(args[2])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	itemID := int(itemIDRaw)
	intimacyPerItem, oppositeGenderOnly, ok := intimacyItemEffect(itemID)
	if !ok || quantity > int(^uint(0)>>1)/intimacyPerItem {
		return nil, pkgerrors.ErrInvalidArgs
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	if h.socialService == nil {
		return nil, fmt.Errorf("intimacy service unavailable")
	}
	sourceChar, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	targetChar, err := h.resolveTargetCharacter(ctx, targetName)
	if err != nil || targetChar == nil || targetChar.ID == characterID {
		_ = ctx.Connection.SendCallback("onSystemSay", "Không tìm thấy đối tượng tặng hợp lệ.")
		return nil, nil
	}
	if oppositeGenderOnly && sourceChar.Gender == targetChar.Gender {
		_ = ctx.Connection.SendCallback("onSystemSay", "Hoa Hồng chỉ có thể tặng cho người khác giới.")
		return nil, nil
	}

	sourceItem, err := h.findBagItemStackByTemplateID(ctx, characterID, itemID, quantity)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	if sourceItem == nil {
		_ = ctx.Connection.SendCallback("onSystemSay", "Không đủ vật phẩm.")
		return nil, nil
	}
	remaining := sourceItem.StackCount - quantity

	amount := quantity * intimacyPerItem
	if _, err := h.socialService.PreviewFriendIntimacy(ctx.Context, characterID, targetChar.ID, amount); err != nil {
		_ = ctx.Connection.SendCallback("onSystemSay", "Chỉ có thể tặng vật phẩm thân mật cho bạn bè.")
		return nil, nil
	}

	consumed, err := h.itemService.ConsumeItemsByTemplateID(ctx.Context, characterID, itemID, quantity)
	if err != nil {
		h.logger.Error("UseIntimacyItem: failed to consume items", zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}
	if !consumed {
		ctx.Connection.SendCallback("onSystemSay", "Không đủ vật phẩm.")
		return nil, nil
	}
	h.sendStackDeltaCallback(ctx, sourceItem, remaining)

	update, err := h.socialService.AddFriendIntimacy(ctx.Context, characterID, targetChar.ID, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to persist intimacy after item consumption; item preservation is not guaranteed: %w", err)
	}
	h.sendRelationshipListRefresh(ctx.Context, ctx.Connection, characterID)
	if h.rtmpServer != nil && update.TargetSaved {
		if targetConn := h.rtmpServer.GetConnectionByCharacterID(strconv.FormatInt(targetChar.ID, 10)); targetConn != nil {
			h.sendRelationshipListRefresh(ctx.Context, targetConn, targetChar.ID)
		}
	}
	_ = ctx.Connection.SendCallback("onBlueMsg", fmt.Sprintf("Đã tặng %s %d %s, tăng %d điểm thân mật.", targetChar.Name, quantity, intimacyItemName(itemID), amount))

	h.logger.Info("UseIntimacyItem: items consumed",
		zap.Int64("character_id", characterID),
		zap.Int("template_id", itemID),
		zap.Int("quantity", quantity),
		zap.Int("intimacy_added", amount))

	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) sendRelationshipListRefresh(ctx context.Context, conn *rtmp.Connection, characterID int64) {
	if h.socialService == nil || conn == nil {
		return
	}
	relationships, err := h.socialService.GetRelationshipList(ctx, characterID)
	if err != nil {
		h.logger.Warn("Failed to refresh relationship list after intimacy gift", zap.Int64("character_id", characterID), zap.Error(err))
		return
	}
	payload := make(map[string]interface{}, len(relationships))
	for _, relationship := range relationships {
		id, ok := relationship["id"].(int64)
		if !ok {
			continue
		}
		cloned := make(map[string]interface{}, len(relationship))
		for key, value := range relationship {
			cloned[key] = value
		}
		if relationshipType, ok := cloned["type"].(int); ok {
			cloned["type"] = relationshipType + 1
		}
		payload[strconv.FormatInt(id, 10)] = cloned
	}
	_ = conn.SendCallback("onInitViewImC", payload)
}

func intimacyItemEffect(templateID int) (int, bool, bool) {
	switch templateID {
	case roseTemplateID:
		return 60, true, true
	case largeRoseTemplateID:
		return 660, true, true
	case chocolateTemplateID:
		return 60, false, true
	case chocolateBoxTemplateID:
		return 660, false, true
	default:
		return 0, false, false
	}
}

func intimacyItemName(templateID int) string {
	switch templateID {
	case roseTemplateID:
		return "Hoa Hồng"
	case largeRoseTemplateID:
		return "Bó Hoa Hồng To"
	case chocolateTemplateID:
		return "Chocolate"
	case chocolateBoxTemplateID:
		return "Hộp Chocolate"
	default:
		return "vật phẩm"
	}
}
