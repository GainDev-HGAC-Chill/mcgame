// Pet skill books (type 506, use_type 2) grant skills to the selected Pet.
package item

import (
	"context"
	"fmt"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
	"strings"
)

type petSkillBookLearner interface {
	LearnBookSkill(ctx context.Context, charID, petID int64, skillID int) (*domainpet.Pet, error)
}

type creBookEffectHandler struct{ itemService *Service }

func (h *creBookEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool {
	return int(tpl.Type) == 506
}

func (h *creBookEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	if h.itemService == nil || h.itemService.petService == nil || h.itemService.gameDataRec == nil || tpl == nil || char == nil {
		return nil, pkgerrors.ErrCannotUseItem
	}
	if int(tpl.UseType) != 2 {
		// Retain the existing creature-contract behavior for non-skill books.
		creatureID := int(tpl.I1)
		if creatureID <= 0 {
			return nil, pkgerrors.ErrCannotUseItem
		}
		pets, err := h.itemService.petService.ContractPets(ctx, char.ID, []int{creatureID})
		if err != nil {
			return nil, err
		}
		result := NewEffectResult()
		result.Pets = append(result.Pets, pets...)
		result.Message = fmt.Sprintf("Đã triệu hồi pet từ %s.", tpl.Name)
		return result, nil
	}
	if uctx == nil || uctx.PetID <= 0 {
		return nil, pkgerrors.ErrCannotUseItem
	}
	learner, ok := h.itemService.petService.(petSkillBookLearner)
	if !ok {
		return nil, pkgerrors.ErrCannotUseItem
	}
	// Exact mappings for verified books. Other 506 books use the level-1 skill name.
	skillID := 0
	switch int(tpl.ID) {
	case 1622:
		skillID = 3590 // Thuốc Kích Thích
	case 1623:
		skillID = 3678 // Quyến Luyến Không Rời
	default:
		for _, candidate := range h.itemService.gameDataRec.GetAllSkills() {
			if candidate == nil || int(candidate.Level) != 1 || int(candidate.Kind) != 1 {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(candidate.Name), strings.TrimSpace(tpl.Name)) {
				continue
			}
			if skillID == 0 || int(candidate.ID) < skillID {
				skillID = int(candidate.ID)
			}
		}
	}
	if skillID == 0 {
		return nil, pkgerrors.ErrCannotUseItem
	}
	pet, err := learner.LearnBookSkill(ctx, char.ID, uctx.PetID, skillID)
	if err != nil {
		return nil, err
	}
	result := NewEffectResult()
	result.UpdatedPets = []*domainpet.Pet{pet}
	result.Message = fmt.Sprintf("Pet đã học %s.", tpl.Name)
	return result, nil
}
