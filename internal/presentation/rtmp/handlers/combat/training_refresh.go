// Open-sourced by BaoLT

package combat

import (
	domainchar "mcgame-server/internal/domain/character"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"

	"go.uber.org/zap"
)

func (h *Handler) isTrainingSkill(skillID int) bool {
	if h == nil || h.gameData == nil {
		return false
	}
	template := h.gameData.GetSkill(skillID)
	return template != nil && int(template.UseEnv) == 3 && int(template.Kind) == 5
}

func (h *Handler) sendTrainingStatRefresh(ctx *rtmp.RPCContext, characterID int64, char *domainchar.Character) {
	if h == nil || ctx == nil || ctx.Connection == nil {
		return
	}
	if char == nil && h.charService != nil {
		loaded, err := h.charService.GetCharacter(ctx.Context, characterID)
		if err != nil {
			h.logger.Warn("Failed to refresh character after Training mutation", zap.Int64("character_id", characterID), zap.Error(err))
		} else {
			char = loaded
		}
	}
	if char != nil {
		rtmputils.SendStatRefreshUPP(ctx, h.itemService, char)
	}
	if h.petService == nil {
		return
	}
	pets, err := h.petService.GetPetList(ctx.Context, characterID)
	if err != nil {
		h.logger.Warn("Failed to refresh Pets after Training mutation", zap.Int64("character_id", characterID), zap.Error(err))
		return
	}
	sendTrainingPetStatRefreshes(ctx.Connection, pets)
}

func sendTrainingPetStatRefreshes(sender skillCallbackSender, pets []*domainpet.Pet) {
	if sender == nil {
		return
	}
	for _, pet := range pets {
		if pet == nil {
			continue
		}
		data, ok := pet.ToDTO()["data"].(map[string]interface{})
		if !ok {
			continue
		}
		property, ok := data["property"]
		if !ok {
			continue
		}
		_ = sender.SendCallback("onRefreshPetProp", map[string]interface{}{"id": pet.ID, "s": property})
	}
}
