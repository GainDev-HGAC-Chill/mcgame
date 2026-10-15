package item

import (
	"strconv"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const tempBagClientPageSize = 24

func (h *Handler) AddTempItemFromBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.addItemFromBagToTemp(ctx, args, false)
}

func (h *Handler) AddMxTempItemFromBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.addItemFromBagToTemp(ctx, args, true)
}

func (h *Handler) TakeTempItemToBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.takeItemFromTempToBag(ctx, args, false)
}

func (h *Handler) TakeMxTempItemToBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.takeItemFromTempToBag(ctx, args, true)
}

func (h *Handler) addItemFromBagToTemp(ctx *rtmp.RPCContext, args []interface{}, mx bool) (interface{}, error) {
	if len(args) < 2 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	sourceSIDRaw, sourceOK := parseFlexibleInt64(args[0])
	targetRaw, targetOK := parseFlexibleInt64(args[1])
	if !sourceOK || !targetOK {
		return nil, pkgerrors.ErrInvalidArgs
	}
	targetPosition, ok := decodeTempBagClientPosition(int(targetRaw))
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
	items, err := h.itemService.GetAllItems(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	payload := h.buildTBagData(char, items)
	capacity := tempBagPayloadCapacity(payload, mx)
	if targetPosition > capacity {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	sourceType, sourceIndex := domainitem.ParseSID(int(sourceSIDRaw))
	if sourceType != domainitem.SlotTypeBag && sourceType != domainitem.SlotTypeQuestBag && sourceType != domainitem.SlotTypePetItemBag {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	source, err := h.itemService.GetItemInSlot(ctx.Context, characterID, sourceType, sourceIndex)
	if err != nil {
		return rtmp.ErrorToResponse(pkgerrors.ErrItemNotFound), nil
	}
	targetIndex := targetPosition
	if mx {
		targetIndex += 100
	}
	existing, _ := h.itemService.GetItemInSlot(ctx.Context, characterID, domainitem.SlotTypeTempBag, targetIndex)
	if existing != nil && !source.CanStack(existing) {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	if err := h.itemService.MoveItem(ctx.Context, characterID, source.ID, domainitem.SlotTypeTempBag, targetIndex); err != nil {
		if pkgerrors.Is(err, pkgerrors.ErrInvalidTempBagItem) {
			if mx {
				_ = ctx.Connection.SendCallback("onMidNote", "Chỉ có thể đặt Ma Tâm.")
			} else {
				_ = ctx.Connection.SendCallback("onMidNote", "Chỉ cần đặt vật phẩm nguyên liện( Bảo Thạch, Lông Vũ, Nguyên Liệu Chế Tạo, Nguyên Liệu Kỹ Năng Sống).")
			}
			return nil, nil
		}
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(source.ID), float64(sourceSIDRaw))
	items, err = h.itemService.GetAllItems(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	payload = h.buildTBagData(char, items)
	_ = ctx.Connection.SendCallback("updateTempBags", payload)
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) takeItemFromTempToBag(ctx *rtmp.RPCContext, args []interface{}, mx bool) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	rawPosition, ok := parseFlexibleInt64(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	position, ok := decodeTempBagClientPosition(int(rawPosition))
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
	tempIndex := position
	if mx {
		tempIndex += 100
	}
	moved, err := h.itemService.MoveTempBagItemToBag(ctx.Context, characterID, tempIndex)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onAddCharactorSlot", h.itemDTO(moved))
	items, err := h.itemService.GetAllItems(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	payload := h.buildTBagData(char, items)
	_ = ctx.Connection.SendCallback("updateTempBags", payload)
	return map[string]interface{}{"success": true}, nil
}

func decodeTempBagClientPosition(raw int) (int, bool) {
	if raw <= 0 {
		return 0, false
	}
	if raw < 100 {
		return raw, true
	}
	page := raw / 100
	position := raw % 100
	if position <= 0 || position > tempBagClientPageSize {
		return 0, false
	}
	return page*tempBagClientPageSize + position, true
}

func tempBagPayloadCapacity(payload map[string]interface{}, mx bool) int {
	if mx {
		mxPayload, ok := payload["mx"].(map[string]interface{})
		if !ok {
			return 0
		}
		capacity, _ := mxPayload["tempSlotNum"].(int)
		return capacity
	}
	capacity, _ := payload["tempSlotNum"].(int)
	return capacity
}
