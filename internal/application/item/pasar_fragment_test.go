package item

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"

	"go.uber.org/zap"
)

func TestUseItemIgnoresPasarFragmentWithoutConsuming(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[88] = &domainitem.Item{
		ID: 88, CharacterID: 99, TemplateID: pasarFragmentTemplateID,
		ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 30,
	}
	service := NewService(repo, zap.NewNop())
	used, result, err := service.UseItem(context.Background(), &domainchar.Character{ID: 99}, 88, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ignored, _ := result["ignored"].(bool); !ignored {
		t.Fatalf("result = %#v", result)
	}
	if used.StackCount != 30 || repo.items[88].StackCount != 30 {
		t.Fatalf("fragment was consumed: used=%+v stored=%+v", used, repo.items[88])
	}
}
