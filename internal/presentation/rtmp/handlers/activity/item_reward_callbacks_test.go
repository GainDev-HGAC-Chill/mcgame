package activity

import (
	"testing"

	appitem "mcgame-server/internal/application/item"
	domainitem "mcgame-server/internal/domain/item"

	"go.uber.org/zap"
)

func TestActivityItemRewardPayloadUsesFlashSlotDTO(t *testing.T) {
	h := NewHandler(appitem.NewService(nil, zap.NewNop()), zap.NewNop())
	added := &domainitem.Item{
		ID:         44,
		TemplateID: 900,
		ItemType:   domainitem.ItemTypeConsumable,
		SlotType:   domainitem.SlotTypeBag,
		SlotIndex:  3,
		StackCount: 5,
		IsBound:    true,
	}

	notice, slot := h.activityItemRewardPayloads(added)
	if notice["i"] != 900 || notice["q"] != 5 || notice["t"] != 29 {
		t.Fatalf("notice = %#v", notice)
	}
	if slot["id"] != int64(44) || slot["sid"] != 2104 || slot["stackNum"] != 5 || slot["binded"] != "1" {
		t.Fatalf("slot = %#v", slot)
	}
	if _, exists := slot["CharacterID"]; exists {
		t.Fatalf("slot leaked Go struct fields: %#v", slot)
	}
}
