package item

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestUseEquipmentReturnsFinalSlotWithDetachedRepository(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[88] = &domainitem.Item{ID: 88, CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeBag, SlotIndex: 15, StackCount: 1}
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadEquipmentTemplate(t, manager, models.EquiptTemplateTemplate{ID: 2001, Position: 10})
	svc.SetGameDataManager(manager)
	it, result, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 99}, 88, nil)
	if err != nil {
		t.Fatal(err)
	}
	if it.SlotType != domainitem.SlotTypeEquipped || it.CalculateSID() != 10 || result["equipped"] != true {
		t.Fatalf("returned stale bag snapshot: %#v result=%v", it, result)
	}
	if repo.items[88].CalculateSID() != it.CalculateSID() || it.StackCount != 1 {
		t.Fatal("equipment state/count mismatch")
	}
}
