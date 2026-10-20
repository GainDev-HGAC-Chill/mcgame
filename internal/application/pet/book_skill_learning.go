// Pet skill books write to the same skill1..skill15 slots used by battles.
package pet

import (
	"context"
	"fmt"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

// LearnBookSkill validates ownership, prevents duplicates and uses only opened slots.
func (s *Service) LearnBookSkill(ctx context.Context, charID, petID int64, skillID int) (*domainpet.Pet, error) {
	if s == nil || s.petRepo == nil || s.gameDataManager == nil || petID <= 0 || skillID <= 0 {
		return nil, pkgerrors.ErrCannotUseItem
	}
	if s.gameDataManager.GetSkill(skillID) == nil {
		return nil, pkgerrors.ErrCannotUseItem
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, pkgerrors.ErrNotFound
	}
	ensurePetSkillSlots(p)
	free := 0
	for slot := 1; slot <= petSkillLastOpenSlot; slot++ {
		existing := petSkillSlotValue(p, slot)
		if existing == skillID {
			return nil, fmt.Errorf("Pet đã học kỹ năng này")
		}
		if existing == 0 && free == 0 {
			free = slot
		}
	}
	if free == 0 {
		return nil, fmt.Errorf("Pet không còn ô kỹ năng đã mở")
	}
	setPetSkillSlotValue(p, free, skillID)
	if err := s.petRepo.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
