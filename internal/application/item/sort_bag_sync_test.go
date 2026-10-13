package item

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestSortBagKeepsQuantitiesAndMakesFinalSlotsMovable(t *testing.T) {
	ctx := context.Background()
	repo := newItemServiceTestRepo()
	repo.items[3] = &domainitem.Item{ID: 3, CharacterID: 99, TemplateID: 125, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 15, StackCount: 20}
	repo.items[8] = &domainitem.Item{ID: 8, CharacterID: 99, TemplateID: 125, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 20, StackCount: 5}
	repo.items[7] = &domainitem.Item{ID: 7, CharacterID: 99, TemplateID: 1009, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeBag, SlotIndex: 22, StackCount: 1, ColorCode: 4}
	repo.items[9] = &domainitem.Item{ID: 9, CharacterID: 99, TemplateID: 125, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 23, StackCount: 4, IsBound: true}
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 125, Name: "Potion", Kind: 5, Type: 501, UseType: 3, StackMax: 99999})
	loadEquipmentTemplate(t, manager, models.EquiptTemplateTemplate{ID: 1009, Name: "Jewelry", Kind: 4, Type: 402})
	svc.SetGameDataManager(manager)
	after, err := svc.SortBag(ctx, 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 3 {
		t.Fatalf("survivors=%d want3", len(after))
	}
	counts := map[int]int{}
	for index, it := range after {
		if it.SlotIndex != index {
			t.Fatalf("index=%d want%d", it.SlotIndex, index)
		}
		stored, err := repo.FindBySlot(ctx, 99, domainitem.SlotTypeBag, index)
		if err != nil || stored.ID != it.ID {
			t.Fatalf("final slot not readable: %v", err)
		}
		counts[it.TemplateID] += it.StackCount
	}
	if counts[125] != 29 || counts[1009] != 1 {
		t.Fatalf("quantities changed: %v", counts)
	}
	if err := svc.MoveItem(ctx, 99, after[0].ID, domainitem.SlotTypeBag, 15); err != nil {
		t.Fatal(err)
	}
	if it, err := repo.FindBySlot(ctx, 99, domainitem.SlotTypeBag, 15); err != nil || it.ID != after[0].ID {
		t.Fatalf("sort-then-move failed: %v", err)
	}
}
