package cache

import (
	"testing"

	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
)

func TestPlayerDataApplyPersistedPetSkillBookIsClean(t *testing.T) {
	data := NewPlayerData(1)
	data.AddPetWithoutDirty(&pet.Pet{ID: 7, CharacterID: 1, Property: map[string]interface{}{"skill1": 2830}})
	data.AddItemWithoutDirty(&item.Item{ID: 9, CharacterID: 1, StackCount: 2})
	updatedPet := &pet.Pet{ID: 7, CharacterID: 1, Property: map[string]interface{}{"skill1": 5527}}

	data.ApplyPersistedPetSkillBook(updatedPet, 9, 1)
	if got := data.GetPet(7); got == nil || got.Property["skill1"] != 5527 {
		t.Fatalf("committed Pet = %#v", got)
	}
	if got := data.GetItem(9); got == nil || got.StackCount != 1 {
		t.Fatalf("committed book = %#v", got)
	}
	if data.IsDirty() {
		t.Fatal("committed skill-book state was marked dirty")
	}

	data.ApplyPersistedPetSkillBook(updatedPet, 9, 0)
	if data.GetItem(9) != nil {
		t.Fatal("consumed book remained in cache")
	}
	if data.IsDirty() {
		t.Fatal("committed deletion was marked dirty")
	}
}
