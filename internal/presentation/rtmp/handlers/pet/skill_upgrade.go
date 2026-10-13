// Pet skill upgrade RPC synchronizes the consumed manual and upgraded slot.
package pet

import (
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) PetUpSkill(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 2); err != nil {
		return nil, err
	}
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	slot, err := parseIntArg(args[1])
	if err != nil {
		return nil, err
	}
	result, err := h.petService.UpgradePetSkill(ctx.Context, characterID, petID, slot)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	if result.ItemDeleted {
		_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(result.ConsumedItemID), float64(result.ConsumedItemSID))
	} else {
		_ = ctx.Connection.SendCallback("onUpChaSlotStackNum", float64(result.ConsumedItemID), float64(result.RemainingStack), float64(result.ConsumedItemSID))
	}
	_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), result.SlotKey, float64(result.SkillID))
	sendPetPropRefresh(ctx, result.Pet)
	return nil, nil
}
