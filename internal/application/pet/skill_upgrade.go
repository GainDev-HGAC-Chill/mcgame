// Pet skill upgrades follow the Flash level chain and consume quality-specific manuals.
package pet

import (
	"context"
	"errors"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

var ErrPetSkillUpgradeItemRequired = errors.New("pet skill upgrade item required")

var petSkillUpgradeItems = map[int]int{
	1: 1365,
	2: 1366,
	3: 1367,
}

type PetSkillUpgradeResult struct {
	Pet             *domainpet.Pet
	SlotKey         string
	SkillID         int
	ConsumedItemID  int64
	ConsumedItemSID int
	RemainingStack  int
	ItemDeleted     bool
}

func (s *Service) UpgradePetSkill(ctx context.Context, charID int64, petID int64, slot int) (*PetSkillUpgradeResult, error) {
	if s.petRepo == nil || s.itemRepo == nil || s.gameDataManager == nil || slot < 1 || slot > domainpet.TotalPetSkillSlots {
		return nil, pkgerrors.ErrInvalidInput
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	ensurePetSkillSlots(p)
	currentSkillID := petSkillSlotValue(p, slot)
	currentSkill := s.gameDataManager.GetSkill(currentSkillID)
	if currentSkill == nil || currentSkill.CodeName == "" || int(currentSkill.Level) < 1 || int(currentSkill.Level) >= 3 {
		return nil, pkgerrors.ErrInvalidInput
	}

	var nextSkillID int
	for _, skill := range s.gameDataManager.GetAllSkills() {
		if skill != nil && skill.CodeName == currentSkill.CodeName && int(skill.Level) == int(currentSkill.Level)+1 {
			nextSkillID = int(skill.ID)
			break
		}
	}
	if nextSkillID <= 0 {
		return nil, pkgerrors.ErrNotFound
	}
	manualTemplateID, ok := petSkillUpgradeItems[s.petSkillQualityLevel(p)]
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	required := 1
	if int(currentSkill.Level) == 2 {
		required = 3
	}
	manual, err := s.findPetSkillUpgradeManual(ctx, charID, manualTemplateID, required)
	if err != nil {
		return nil, err
	}

	originalPet := clonePetSnapshot(p)
	setPetSkillSlotValue(p, slot, nextSkillID)
	s.syncPetSkillBonuses(p)
	p.RecalculateStats()
	if err := s.petRepo.Save(ctx, p); err != nil {
		return nil, err
	}
	remaining := manual.StackCount - required
	deleted := remaining == 0
	if deleted {
		err = s.itemRepo.Delete(ctx, manual.ID)
	} else {
		err = s.itemRepo.UpdateStack(ctx, manual.ID, remaining)
	}
	if err != nil {
		if rollbackErr := s.petRepo.Save(ctx, originalPet); rollbackErr != nil && s.logger != nil {
			s.logger.Error("Failed to rollback pet skill upgrade", zap.Int64("pet_id", petID), zap.Error(rollbackErr))
		}
		return nil, err
	}
	manual.StackCount = remaining
	return &PetSkillUpgradeResult{
		Pet:             p,
		SlotKey:         petSkillSlotKey(slot),
		SkillID:         nextSkillID,
		ConsumedItemID:  manual.ID,
		ConsumedItemSID: manual.CalculateSID(),
		RemainingStack:  remaining,
		ItemDeleted:     deleted,
	}, nil
}

func (s *Service) findPetSkillUpgradeManual(ctx context.Context, charID int64, templateID int, required int) (*domainitem.Item, error) {
	for _, slotType := range []domainitem.SlotType{domainitem.SlotTypeBag, domainitem.SlotTypePetItemBag} {
		items, err := s.itemRepo.FindByCharacterAndSlotType(ctx, charID, slotType)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if it != nil && it.TemplateID == templateID && it.StackCount >= required {
				return it, nil
			}
		}
	}
	return nil, ErrPetSkillUpgradeItemRequired
}
