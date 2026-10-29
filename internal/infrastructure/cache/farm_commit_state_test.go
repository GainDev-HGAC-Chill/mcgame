package cache

import (
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

func TestSyncCommittedFarmStateReconcilesCharacterAndItemsWithoutDirtyWrites(t *testing.T) {
	data := NewPlayerData(3)
	data.Character = &domainchar.Character{ID: 3, CurrentSP: 100, Gold: 100}
	data.Items[10] = &domainitem.Item{ID: 10, CharacterID: 3, TemplateID: 2909, StackCount: 1}
	data.dirtyChar = true
	data.dirtyItems[10] = true
	data.dirtyItems[20] = true
	data.deletedItems = []int64{20}

	data.syncCommittedFarmState(
		&domainchar.Character{ID: 3, CurrentSP: 75, Gold: 91},
		&domainitem.Item{ID: 10, CharacterID: 3, TemplateID: 2909, StackCount: 0},
		&domainitem.Item{ID: 20, CharacterID: 3, TemplateID: 2481, StackCount: 11},
	)

	if data.Character.CurrentSP != 75 || data.Character.Gold != 91 || data.Items[10] != nil || data.Items[20] == nil {
		t.Fatalf("committed farm state not reconciled: character=%+v items=%+v", data.Character, data.Items)
	}
	if data.IsDirty() {
		t.Fatalf("committed farm state remained dirty: %+v", data.GetDirtySnapshot())
	}
}
