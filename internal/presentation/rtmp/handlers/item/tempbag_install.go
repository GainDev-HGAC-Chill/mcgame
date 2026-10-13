package item

import (
	"strconv"
	"time"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) SetTempBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil || characterID <= 0 {
		return nil, pkgerrors.ErrUnauthorized
	}
	if len(args) != 2 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	fromSID, fromOK := parseFlexibleInt64(args[0])
	toSID, toOK := parseFlexibleInt64(args[1])
	if !fromOK || !toOK || fromSID < 2101 || fromSID > 2310 || toSID < 80 || toSID > 84 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	result, err := h.itemService.InstallTempBag(ctx.Context, characterID, int(fromSID), int(toSID), char.TempBagSlots, time.Now())
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(result.Installed.ID), float64(fromSID))
	if result.Replaced != nil {
		_ = ctx.Connection.SendCallback("onAddCharactorSlot", h.itemDTO(result.Replaced))
	}
	items, err := h.itemService.GetAllItems(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	payload := h.buildTBagData(char, items)
	_ = ctx.Connection.SendCallback("updateTempBags", payload)
	return payload, nil
}
