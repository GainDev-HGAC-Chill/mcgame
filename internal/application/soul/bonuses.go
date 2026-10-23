// Open-sourced by BaoLT

// Equipped Pet Soul rows contribute their exact catalog property values to Pet stats.
package soul

import (
	"context"
	"math"

	domainpet "mcgame-server/internal/domain/pet"
)

func (s *Service) AggregatePetBonuses(ctx context.Context, petID int64) (domainpet.EquipmentStatBonuses, domainpet.SkillStatBonuses) {
	equipment := domainpet.NewEquipmentStatBonuses()
	skills := domainpet.NewSkillStatBonuses()
	if s == nil || s.petRepo == nil || s.gameData == nil {
		return equipment, skills
	}
	pet, err := s.petRepo.FindByID(ctx, petID)
	if err != nil || pet == nil {
		return equipment, skills
	}
	unlock := s.lockMutation(pet.CharacterID)
	defer unlock()
	state, err := s.loadPetState(ctx, petID)
	if err != nil {
		return domainpet.NewEquipmentStatBonuses(), skills
	}
	for _, slot := range state.Slots {
		tpl := s.petSoulTemplate(slot.Sid)
		if tpl == nil || tpl.PropType <= 0 || tpl.PropType != math.Trunc(tpl.PropType) || math.IsNaN(tpl.PropVal) || math.IsInf(tpl.PropVal, 0) {
			return domainpet.NewEquipmentStatBonuses(), skills
		}
		propType := int(tpl.PropType)
		if isIntegralPetSoulProperty(propType) && tpl.PropVal == math.Trunc(tpl.PropVal) {
			equipment.AddFlat(propType, int(tpl.PropVal))
			continue
		}
		equipment.AddFeature(propType, tpl.PropVal)
	}
	return equipment, skills
}

func isIntegralPetSoulProperty(propType int) bool {
	switch propType {
	case 1, 4, 5, 6, 7, 8, 9, 10, 11, 13:
		return true
	default:
		return false
	}
}
