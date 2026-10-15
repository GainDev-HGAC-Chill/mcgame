// Open-sourced by BaoLT

package item

import (
	"context"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

var petGrowthMedicineExperience = map[int]int64{
	961:  30000,
	962:  150000,
	964:  300000,
	1585: 300000,
	1586: 2000000,
	1587: 6400000,
	1836: 12500000,
	3339: 100000000,
}

type petExperienceService interface {
	GetPetDetail(ctx context.Context, charID int64, petID int64) (*domainpet.Pet, error)
	GainExperience(ctx context.Context, charID int64, petID int64, amount int64) (*domainpet.Pet, bool, error)
}

type petGrowthOrPotionEffectHandler struct {
	itemService *Service
}

func (h *petGrowthOrPotionEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool {
	return int(tpl.Type) == 501
}

func (h *petGrowthOrPotionEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	experience, growthMedicine := petGrowthMedicineExperience[int(tpl.ID)]
	if !growthMedicine {
		return (&potionEffectHandler{}).Apply(ctx, uctx, char, it, tpl)
	}
	if uctx == nil || uctx.TargetType != 2 || uctx.PetID <= 0 {
		return nil, pkgerrors.ErrCannotUseItem
	}
	petService, ok := h.itemService.petService.(petExperienceService)
	if !ok {
		return nil, pkgerrors.ErrCannotUseItem
	}
	pet, err := petService.GetPetDetail(ctx, char.ID, uctx.PetID)
	if err != nil {
		return nil, err
	}
	if pet.Level < int(tpl.ReqLevel) {
		return nil, pkgerrors.ErrInsufficientLevel
	}
	if pet.Level >= domainpet.MaxPetLevel {
		return nil, pkgerrors.ErrCannotUseItem
	}
	quantity := uctx.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	updated, leveledUp, err := petService.GainExperience(ctx, char.ID, pet.ID, experience*int64(quantity))
	if err != nil {
		return nil, err
	}
	result := NewEffectResult()
	result.PetProgress = []PetProgressResult{{Pet: updated, LeveledUp: leveledUp}}
	return result, nil
}

func isPetGrowthMedicine(templateID int) bool {
	_, ok := petGrowthMedicineExperience[templateID]
	return ok
}
