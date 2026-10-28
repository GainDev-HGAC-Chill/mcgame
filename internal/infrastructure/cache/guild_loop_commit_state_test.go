package cache

import (
	"testing"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	domainquest "mcgame-server/internal/domain/quest"
)

func TestSyncCommittedGuildLoopReconcilesWithoutDirtyWrites(t *testing.T) {
	data := NewPlayerData(3)
	data.Character = &domainchar.Character{ID: 3, Money: 1, MoneyBind: 2}
	data.Items[10] = &domainitem.Item{ID: 10, StackCount: 5}
	data.Items[11] = &domainitem.Item{ID: 11, StackCount: 1}
	data.Pets[20] = &domainpet.Pet{ID: 20}
	data.Quests[30] = &domainquest.QuestProgress{ID: 30, CharacterID: 3, QuestID: 100}
	data.dirtyItems[10] = true
	data.dirtyItems[11] = true
	data.dirtyPets[20] = true
	data.dirtyQuests[30] = true
	data.deletedItems = []int64{10, 11}
	data.deletedPets = []int64{20}
	data.deletedQuests = []int64{30}
	next := &domainquest.QuestProgress{ID: 31, CharacterID: 3, QuestID: 101, StartedAt: time.Now()}
	result := &domainquest.GuildLoopCompletionResult{
		GuildID: 1, Money: 100, MoneyBind: 200, Gold: 3, GuildContrib: 40,
		ItemStates:     []domainquest.GuildLoopItemState{{ItemID: 10, Remaining: 3}, {ItemID: 11, Remaining: 0}},
		ConsumedPetIDs: []int64{20}, NextQuest: next,
	}

	data.syncCommittedGuildLoop(30, result)

	if data.Character.Money != 100 || data.Character.MoneyBind != 200 || data.Character.GuildContrib != 40 || data.Character.GuildID == nil || *data.Character.GuildID != 1 {
		t.Fatalf("character not reconciled: %+v", data.Character)
	}
	if data.Items[10] == nil || data.Items[10].StackCount != 3 || data.Items[11] != nil {
		t.Fatalf("items not reconciled: %+v", data.Items)
	}
	if data.Pets[20] != nil || data.Quests[30] != nil || data.Quests[31] == nil {
		t.Fatalf("pet/quest state not reconciled: pets=%+v quests=%+v", data.Pets, data.Quests)
	}
	if data.IsDirty() {
		t.Fatalf("committed state remained dirty: %+v", data.GetDirtySnapshot())
	}
}
