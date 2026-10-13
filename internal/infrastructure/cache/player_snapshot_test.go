package cache

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	"mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/skill"
)

func TestPlayerDataDetachedEntitiesAndSnapshots(t *testing.T) {
	p := NewPlayerData(1)
	char := &character.Character{ID: 1, BossDaily: map[string]interface{}{"data": map[string]interface{}{"n": int64(7)}}}
	pt := &pet.Pet{ID: 2, CharacterID: 1, Property: map[string]interface{}{"values": []int64{7}}, SoulInfo: map[string]interface{}{"data": map[string]interface{}{"n": 7}}}
	durability := 7
	it := &item.Item{ID: 3, Properties: map[string]interface{}{"n": 7}, Durability: &durability}
	slot := 7
	sk := &skill.CharacterSkill{ID: 4, SlotPosition: &slot}
	now := time.Now()
	qp := &quest.QuestProgress{ID: 5, Objectives: []quest.Objective{{Current: 7}}, CompletedAt: &now}
	p.SetCharacter(char)
	p.SetPet(pt)
	p.SetItem(it)
	p.SetSkill(sk)
	p.SetQuest(qp)
	char.BossDaily["data"].(map[string]interface{})["n"] = int64(99)
	pt.Property["values"].([]int64)[0] = 99
	pt.SoulInfo["data"].(map[string]interface{})["n"] = 99
	durability, slot = 99, 99
	qp.Objectives[0].Current = 99
	if p.GetCharacter().BossDaily["data"].(map[string]interface{})["n"] != int64(7) || p.GetPet(2).Property["values"].([]int64)[0] != 7 || *p.GetItem(3).Durability != 7 || *p.GetSkill(4).SlotPosition != 7 || p.GetQuest(5).Objectives[0].Current != 7 {
		t.Fatal("cache retained caller-owned mutable state")
	}
	snapshot, dirty := p.Snapshot(), p.GetDirtySnapshot()
	for _, pt := range []*pet.Pet{p.GetPet(2), p.GetPets()[0]} {
		pt.Property["values"].([]int64)[0] = 80
		pt.SoulInfo["data"].(map[string]interface{})["n"] = 80
	}
	for _, it := range []*item.Item{p.GetItem(3), p.GetItems()[0]} {
		*it.Durability = 80
	}
	for _, sk := range []*skill.CharacterSkill{p.GetSkill(4), p.GetSkills()[0]} {
		*sk.SlotPosition = 80
	}
	for _, qp := range []*quest.QuestProgress{p.GetQuest(5), p.GetQuests()[0]} {
		qp.Objectives[0].Current = 80
	}
	updated := p.GetPet(2)
	updated.Property["values"].([]int64)[0] = 42
	p.SetPet(updated)
	if snapshot.Pets[2].Property["values"].([]int64)[0] != 7 || dirty.DirtyPets[2].Property["values"].([]int64)[0] != 7 {
		t.Fatal("persistence snapshot changed after cache update")
	}
	if *p.GetItem(3).Durability != 7 || *p.GetSkill(4).SlotPosition != 7 || p.GetQuest(5).Objectives[0].Current != 7 || p.GetPet(2).SoulInfo["data"].(map[string]interface{})["n"] != 7 {
		t.Fatal("cache reads leaked mutable state")
	}
}

func TestPlayerDataConcurrentPetMutationAndPersistenceSerialization(t *testing.T) {
	p := NewPlayerData(1)
	pt := &pet.Pet{ID: 1, Property: map[string]interface{}{"state": 1}, SoulInfo: map[string]interface{}{"data": map[string]interface{}{"n": 0}}}
	p.SetPet(pt)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			pt.Property["state"] = i
			pt.SoulInfo["data"].(map[string]interface{})["n"] = i
			p.SetPet(pt)
			copy := p.GetPet(1)
			copy.SetBinded(true)
			copy.SetClientState(5)
			p.SetPet(copy)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			if _, err := json.Marshal(p.Snapshot()); err != nil {
				t.Error(err)
				return
			}
			if _, err := json.Marshal(p.GetDirtySnapshot()); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	wg.Wait()
}

func TestSnapshotAcknowledgementKeepsConcurrentUpdatesDirty(t *testing.T) {
	p := NewPlayerData(1)
	p.SetCharacter(&character.Character{ID: 1, Money: 7})
	snapshot := p.GetDirtySnapshot()
	p.SetCharacter(&character.Character{ID: 1, Money: 42})
	p.acknowledgeSnapshot(snapshot)
	if !p.IsDirty() || p.GetCharacter().Money != 42 {
		t.Fatal("autosave acknowledged an update that was not persisted")
	}
	p.acknowledgeSnapshot(p.GetDirtySnapshot())
	if p.IsDirty() {
		t.Fatal("unchanged saved batch should be clean")
	}
}

func TestSnapshotAcknowledgementDoesNotRetrySavedDeletions(t *testing.T) {
	p := NewPlayerData(1)
	p.DeleteItem(7)
	p.DeletePet(8)
	snapshot := p.GetDirtySnapshot()
	p.SetCharacter(&character.Character{ID: 1, Money: 42})
	p.DeleteItem(9)
	p.acknowledgeSnapshot(snapshot)
	remaining := p.GetDirtySnapshot()
	if len(remaining.DeletedItems) != 1 || remaining.DeletedItems[0] != 9 || len(remaining.DeletedPets) != 0 || !remaining.DirtyChar {
		t.Fatalf("saved tombstones retried or concurrent work lost: %#v", remaining)
	}
}

func TestCachedCharacterPositionAndStatsSaveDetachedCopy(t *testing.T) {
	c, _, _, _, _, _ := newTestPlayerCache(t)
	p := NewPlayerData(1)
	p.SetCharacter(&character.Character{ID: 1})
	c.activeSessions[1] = p
	r := NewCachedCharacterRepository(c, c.charRepo)
	if err := r.UpdatePosition(context.Background(), 1, character.Position{MapID: 3, X: 10, Y: 20}); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateStats(context.Background(), 1, 40, 50, 60); err != nil {
		t.Fatal(err)
	}
	char := p.GetCharacter()
	if char.MapID != 3 || char.PosX != 10 || char.PosY != 20 || char.CurrentHP != 40 || char.CurrentMP != 50 || char.CurrentSP != 60 {
		t.Fatalf("detached mutation lost: %#v", char)
	}
}
