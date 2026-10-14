// Convenience shop handlers bridge legacy fire-and-forget Flash RPCs.
package shop

import (
	"strconv"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) ClickShop(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 || h.gameData == nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	npcID, ok := convenienceInt64(args[0])
	if !ok || npcID <= 0 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	shopID, ok := resolveNPCShopID(h.gameData, int(npcID))
	if !ok {
		return nil, pkgerrors.ErrNotFound
	}
	if err := ctx.Connection.SendCallbackSync("onOpenShop", shopID); err != nil {
		return nil, err
	}
	return nil, nil
}

func resolveNPCShopID(gameData *gamedata.Manager, npcID int) (int, bool) {
	if gameData == nil || npcID <= 0 {
		return 0, false
	}
	npc := gameData.GetNPC(npcID)
	if npc == nil || npc.ShopID <= 0 {
		return 0, false
	}
	return int(npc.ShopID), true
}

func (h *Handler) BuyBloodBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 || h.bloodBag == nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	payload, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	bagType, okType := convenienceInt64(payload["bt"])
	amount, okAmount := convenienceInt64(payload["bn"])
	if !okType || !okAmount {
		return nil, pkgerrors.ErrInvalidArgs
	}
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}
	result, err := h.bloodBag.Purchase(ctx.Context, characterID, int(bagType), amount)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onUPP", map[string]interface{}{"gold": result.NewGold})
	return nil, nil
}

func convenienceInt64(value interface{}) (int64, bool) {
	switch number := value.(type) {
	case float64:
		converted := int64(number)
		return converted, float64(converted) == number
	case int:
		return int64(number), true
	case int64:
		return number, true
	default:
		return 0, false
	}
}
