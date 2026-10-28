package cache

import (
	"testing"

	domainactivity "mcgame-server/internal/domain/activity"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

func TestSyncCommittedSendCombineReconcilesWithoutDirtyWrites(t *testing.T) {
	data := NewPlayerData(3)
	data.Character = &domainchar.Character{ID: 3, Gold: 100}
	data.dirtyChar = true
	committed := &domainitem.Item{ID: 90, CharacterID: 3, TemplateID: 6861, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 4, StackCount: 10, IsBound: true, Properties: map[string]interface{}{}}
	data.dirtyItems[90] = true
	data.deletedItems = []int64{90}

	data.syncCommittedSendCombine(&domainactivity.SendCombinePurchaseResult{BuyCount: 1, Gold: 50, Items: []*domainitem.Item{committed}})

	if data.Character.Gold != 50 || data.Items[90] == nil || data.Items[90].StackCount != 10 {
		t.Fatalf("committed state not reconciled: character=%+v items=%+v", data.Character, data.Items)
	}
	if data.IsDirty() {
		t.Fatalf("committed state remained dirty: %+v", data.GetDirtySnapshot())
	}
}
