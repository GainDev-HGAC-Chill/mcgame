package pet

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"

	"go.uber.org/zap"
)

func TestUpgradePetSkillConsumesQualityManualAndUpdatesSlot(t *testing.T) {
	petRepo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{
		2484410: {
			ID: 2484410, CharacterID: 1, TemplateID: 1001, GrowRate: 1,
			CreatureData: map[string]interface{}{"qLevel": 3},
			Property:     map[string]interface{}{"skill1": 5527},
		},
	}}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		9001: {ID: 9001, CharacterID: 1, TemplateID: 1367, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 2},
	}}
	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(newPetSkillBookGameData(t))
	result, err := service.UpgradePetSkill(context.Background(), 1, 2484410, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.SkillID != 5528 || result.RemainingStack != 1 || result.ItemDeleted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := petSkillSlotValue(petRepo.pets[2484410], 1); got != 5528 {
		t.Fatalf("skill1=%d", got)
	}
	if itemRepo.items[9001].StackCount != 1 {
		t.Fatalf("manual stack=%d", itemRepo.items[9001].StackCount)
	}
}
