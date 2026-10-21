// Pet skill-book persistence uses an atomic repository when available.
package pet

import (
	"context"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"

	"go.uber.org/zap"
)

func (s *Service) persistPetSkillBook(ctx context.Context, originalPet *domainpet.Pet, updatedPet *domainpet.Pet, bookItem *domainitem.Item) (int, error) {
	if repo, ok := s.petRepo.(domainpet.SkillBookRepository); ok {
		return repo.ApplySkillBook(ctx, domainpet.SkillBookMutation{
			Pet:                updatedPet,
			BookItemID:         bookItem.ID,
			ExpectedBookStack:  bookItem.StackCount,
			ExpectedSkillSlots: petSkillSlotSnapshot(originalPet),
		})
	}

	if err := s.petRepo.Save(ctx, updatedPet); err != nil {
		return bookItem.StackCount, err
	}
	remaining, err := s.consumePetSkillBookItem(ctx, bookItem)
	if err == nil {
		return remaining, nil
	}
	if rollbackErr := s.petRepo.Save(ctx, originalPet); rollbackErr != nil && s.logger != nil {
		s.logger.Error("Failed to rollback pet after petBook consume error",
			zap.Int64("character_id", updatedPet.CharacterID),
			zap.Int64("pet_id", updatedPet.ID),
			zap.Int64("item_id", bookItem.ID),
			zap.Error(rollbackErr))
	}
	return bookItem.StackCount, err
}

func petSkillSlotSnapshot(p *domainpet.Pet) []int {
	slots := make([]int, domainpet.TotalPetSkillSlots)
	for slot := 1; slot <= domainpet.TotalPetSkillSlots; slot++ {
		slots[slot-1] = petSkillSlotValue(p, slot)
	}
	return slots
}
