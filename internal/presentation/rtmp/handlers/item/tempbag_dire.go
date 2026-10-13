package item

import (
    "strconv"

    "mcgame-server/internal/infrastructure/rtmp"
    pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) SetTbagDire(
    ctx *rtmp.RPCContext,
    args []interface{},
) (interface{}, error) {
    if len(args) != 1 {
        return nil, pkgerrors.ErrInvalidArgs
    }

    selected, ok := args[0].(bool)
    if !ok {
        return nil, pkgerrors.ErrInvalidArgs
    }

    charID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
    if err != nil || charID <= 0 {
        return nil, pkgerrors.ErrUnauthorized
    }

    char, err := h.charService.GetByID(ctx.Context, charID)
    if err != nil {
        return nil, err
    }

    if char.PMProcessData == nil {
        char.PMProcessData = map[string]interface{}{}
    }

    char.PMProcessData["tempBagDire"] = selected

    if err := h.charService.Update(ctx.Context, char); err != nil {
        return nil, err
    }

    items, err := h.itemService.GetAllItems(ctx.Context, charID)
    if err != nil {
        return nil, err
    }

    payload := h.buildTBagData(char, items)

    if err := ctx.Connection.SendCallback(
        "updateTempBags", payload,
    ); err != nil {
        return nil, err
    }

    return payload["dire"], nil
}