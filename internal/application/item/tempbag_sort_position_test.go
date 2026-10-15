package item

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"

	"go.uber.org/zap"
)

func TestSortTempBagsKeepClientPositionsOneBased(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[1] = &domainitem.Item{ID: 1, CharacterID: 1, TemplateID: 20, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 9, StackCount: 1}
	repo.items[2] = &domainitem.Item{ID: 2, CharacterID: 1, TemplateID: 10, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 3, StackCount: 1}
	repo.items[3] = &domainitem.Item{ID: 3, CharacterID: 1, TemplateID: 40, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 109, StackCount: 1}
	repo.items[4] = &domainitem.Item{ID: 4, CharacterID: 1, TemplateID: 30, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 103, StackCount: 1}
	service := NewService(repo, zap.NewNop())

	if _, err := service.SortTempBag(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if repo.items[2].SlotIndex != 1 || repo.items[1].SlotIndex != 2 {
		t.Fatalf("normal positions = %d/%d, want 1/2", repo.items[2].SlotIndex, repo.items[1].SlotIndex)
	}
	if _, err := service.SortMxTempBag(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if repo.items[4].SlotIndex != 101 || repo.items[3].SlotIndex != 102 {
		t.Fatalf("mx positions = %d/%d, want 101/102", repo.items[4].SlotIndex, repo.items[3].SlotIndex)
	}
}
