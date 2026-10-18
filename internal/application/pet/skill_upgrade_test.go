package pet

import (
	"context"
	"encoding/json"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"

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

func TestUpgradePetSkillRepairsLegacyManLucCollision(t *testing.T) {
	petRepo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{
		2484410: {
			ID: 2484410, CharacterID: 1, TemplateID: 1001, GrowRate: 1,
			CreatureData: map[string]interface{}{"qLevel": 3},
			Property:     map[string]interface{}{"skill4": 60903},
		},
	}}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		9001: {ID: 9001, CharacterID: 1, TemplateID: 1367, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1},
	}}
	manager := newPetSkillBookGameData(t)
	if err := manager.GetCache().LoadTable(models.TableSkill, []json.RawMessage{
		json.RawMessage(`{"id":60903,"name":"Man Lực","code_name":"SKILL1010703","kind":2,"level":1,"use_env":-1}`),
		json.RawMessage(`{"id":4172,"name":"Man Lực","code_name":"SKILL505701","kind":2,"level":1,"use_env":1}`),
		json.RawMessage(`{"id":4171,"name":"Man Lực","code_name":"SKILL505701","kind":2,"level":2,"use_env":1}`),
	}); err != nil {
		t.Fatal(err)
	}
	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(manager)

	result, err := service.UpgradePetSkill(context.Background(), 1, 2484410, 4)
	if err != nil {
		t.Fatal(err)
	}
	if result.SkillID != 4171 || petSkillSlotValue(petRepo.pets[2484410], 4) != 4171 {
		t.Fatalf("legacy Man Lực upgrade = %+v, stored=%d; want 4171", result, petSkillSlotValue(petRepo.pets[2484410], 4))
	}
}
