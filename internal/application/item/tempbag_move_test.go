package item

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"

	"go.uber.org/zap"
)

func TestMoveTempBagItemToBagUsesFirstEmptyRegularSlot(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[1] = &domainitem.Item{ID: 1, CharacterID: 7, TemplateID: 20, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
	repo.items[2] = &domainitem.Item{ID: 2, CharacterID: 7, TemplateID: 30, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 1, StackCount: 4}
	service := NewService(repo, zap.NewNop())

	moved, err := service.MoveTempBagItemToBag(context.Background(), 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if moved.ID != 2 || moved.SlotType != domainitem.SlotTypeBag || moved.SlotIndex != 1 || moved.CalculateSID() != 2102 {
		t.Fatalf("moved = %+v", moved)
	}
	if repo.items[2].SlotType != domainitem.SlotTypeBag || repo.items[2].SlotIndex != 1 {
		t.Fatalf("stored = %+v", repo.items[2])
	}
}

func TestMoveMxTempBagItemToBagUsesEncodedIndex(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[2] = &domainitem.Item{ID: 2, CharacterID: 7, TemplateID: 5676, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 101, StackCount: 4}
	service := NewService(repo, zap.NewNop())

	moved, err := service.MoveTempBagItemToBag(context.Background(), 7, 101)
	if err != nil {
		t.Fatal(err)
	}
	if moved.SlotType != domainitem.SlotTypeBag || moved.SlotIndex != 0 || moved.CalculateSID() != 2101 {
		t.Fatalf("moved = %+v", moved)
	}
}
