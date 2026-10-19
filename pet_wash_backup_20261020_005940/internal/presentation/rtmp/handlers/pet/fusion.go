// Open-sourced by BaoLT

package pet

import (
	"errors"

	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) PetJoin(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 5); err != nil {
		return nil, err
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	mainPetID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	materialPetIDs := make([]int64, 0, 4)
	for _, arg := range args[1:5] {
		materialPetID, err := parseInt64Arg(arg)
		if err != nil {
			return nil, err
		}
		materialPetIDs = append(materialPetIDs, materialPetID)
	}

	result, err := h.petService.FusePets(ctx.Context, characterID, mainPetID, materialPetIDs)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	for _, petID := range result.ConsumedPetIDs {
		_ = ctx.Connection.SendCallback("onDelPet", petID)
	}
	if result.Success {
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "growRate", result.Pet.GrowRate)
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "growRateAdd", result.Pet.GrowRateAdd)
		sendPetPropRefresh(ctx, result.Pet)
	}

	h.logger.Info("Pets fused",
		zap.Int64("character_id", characterID),
		zap.Int64("main_pet_id", mainPetID),
		zap.Int64s("material_pet_ids", materialPetIDs),
		zap.Bool("success", result.Success))

	return map[string]interface{}{
		"flag": result.Success,
		"pid":  result.Pet.ID,
		"grow": result.Pet.GrowRate,
	}, nil
}

func (h *Handler) SeniorPetJoin(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 2); err != nil {
		return nil, err
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	mainPetID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	materialPetIDs := parseInt64SliceArg(args[1])

	p, err := h.petService.SeniorFusePets(ctx.Context, characterID, mainPetID, materialPetIDs)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	for _, petID := range materialPetIDs {
		_ = ctx.Connection.SendCallback("onDelPet", petID)
	}
	_ = ctx.Connection.SendCallback("onUpdatePet", float64(p.ID), "growRate", p.GrowRate+p.GrowRateAdd)

	h.logger.Info("Senior pet fusion completed",
		zap.Int64("character_id", characterID),
		zap.Int64("main_pet_id", mainPetID),
		zap.Int("material_count", len(materialPetIDs)))

	return map[string]interface{}{
		"flag": true,
		"pid":  p.ID,
		"grow": p.GrowRate + p.GrowRateAdd,
	}, nil
}

func (h *Handler) PetXd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
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
	itemID, err := parseInt64Arg(args[1])
	if err != nil {
		return nil, err
	}

	result, err := h.petService.CultivateWithItem(ctx.Context, characterID, petID, itemID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet cultivation performed",
		zap.Int64("character_id", characterID),
		zap.Int64("pet_id", petID),
		zap.Int64("item_id", itemID))
	sendPetItemUseRefresh(ctx, result)

	return map[string]interface{}{
		"f":   true,
		"pet": result.Pet.ToDetailDTO(),
	}, nil
}

func (h *Handler) PetPzXsd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
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
	itemID, err := parseInt64Arg(args[1])
	if err != nil {
		return nil, err
	}

	result, err := h.petService.UseCultivationItem(ctx.Context, characterID, petID, itemID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet cultivation item used",
		zap.Int64("character_id", characterID),
		zap.Int64("pet_id", petID),
		zap.Int64("item_id", itemID))
	sendPetItemUseRefresh(ctx, result)

	return map[string]interface{}{
		"f":      true,
		"p":      result.Pet.ID,
		"t":      result.Property,
		"num":    result.After,
		"i":      result.ItemID,
		"n":      result.RemainingStack,
		"before": result.Before,
		"after":  result.After,
	}, nil
}

func (h *Handler) PetXsd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
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

	itemID, err := parseInt64Arg(args[1])
	if err != nil {
		return nil, err
	}
	result, err := h.petService.PetXsd(ctx.Context, characterID, petID, itemID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet XSD performed",
		zap.Int64("character_id", characterID),
		zap.Int64("pet_id", petID))

	sendPetItemUseRefresh(ctx, result)
	return map[string]interface{}{
		"f":      true,
		"p":      result.Pet.ID,
		"t":      result.Property,
		"num":    result.After,
		"i":      result.ItemID,
		"n":      result.RemainingStack,
		"before": result.Before,
		"after":  result.After,
	}, nil
}

func (h *Handler) PetBook(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 2); err != nil {
		return map[string]interface{}{"f": apppet.PetSkillBookFailureInput}, nil
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return map[string]interface{}{"f": apppet.PetSkillBookFailureInput}, nil
	}
	skillBookItemID, err := parseInt64Arg(args[1])
	if err != nil {
		return map[string]interface{}{"f": apppet.PetSkillBookFailureInput}, nil
	}

	result, err := h.petService.LearnSkillFromBook(ctx.Context, characterID, petID, skillBookItemID)
	if err != nil {
		var bookErr *apppet.PetSkillBookError
		if errors.As(err, &bookErr) {
			response := map[string]interface{}{
				"f": bookErr.Failure,
			}
			if bookErr.Failure == apppet.PetSkillBookFailureLearn {
				response["rf"] = false
			}
			if bookErr.Remaining >= 0 {
				response["n"] = bookErr.Remaining
			}
			return response, nil
		}
		return rtmp.ErrorToResponse(err), nil
	}

	if result.RemainingStack == 0 {
		_ = ctx.Connection.SendCallback(
			"onDelCharactorSlot",
			float64(result.ConsumedItemID),
			float64(result.ConsumedItemSID),
		)
	} else {
		_ = ctx.Connection.SendCallback(
			"onUpChaSlotStackNum",
			float64(result.ConsumedItemID),
			float64(result.RemainingStack),
			float64(result.ConsumedItemSID),
		)
	}

	_ = ctx.Connection.SendCallback(
		"onUpdatePet",
		float64(result.Pet.ID),
		result.SlotKey,
		float64(result.SkillID),
	)
	sendPetPropRefresh(ctx, result.Pet)
	h.logger.Info("Pet learned skill from book",
		zap.Int64("character_id", characterID),
		zap.Int64("pet_id", petID),
		zap.Int64("skill_book_item_id", skillBookItemID),
		zap.Int("skill_id", result.SkillID),
		zap.String("slot", result.SlotKey))

	return map[string]interface{}{
		"f":  1,
		"rf": true,
		"n":  result.RemainingStack,
	}, nil
}
