// Open-sourced by BaoLT

// PlayerData holds all cached entities for a single player/character.
// Provides dirty tracking for write-behind persistence.
// Thread-safe via sync.RWMutex for concurrent RPC access.
package cache

import (
	"slices"
	"sync"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	"mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/skill"
)

type PlayerData struct {
	CharacterID int64
	Character   *character.Character
	Items       map[int64]*item.Item
	Skills      map[int64]*skill.CharacterSkill
	Pets        map[int64]*pet.Pet
	Quests      map[int64]*quest.QuestProgress

	dirtyChar   bool
	dirtyItems  map[int64]bool
	dirtySkills map[int64]bool
	dirtyPets   map[int64]bool
	dirtyQuests map[int64]bool

	deletedItems  []int64
	deletedSkills []int64
	deletedPets   []int64
	deletedQuests []int64

	mu        sync.RWMutex
	persistMu sync.Mutex
	redisMu   sync.Mutex
	revision  uint64
}

func NewPlayerData(charID int64) *PlayerData {
	return &PlayerData{
		CharacterID: charID,
		Items:       make(map[int64]*item.Item),
		Skills:      make(map[int64]*skill.CharacterSkill),
		Pets:        make(map[int64]*pet.Pet),
		Quests:      make(map[int64]*quest.QuestProgress),
		dirtyItems:  make(map[int64]bool),
		dirtySkills: make(map[int64]bool),
		dirtyPets:   make(map[int64]bool),
		dirtyQuests: make(map[int64]bool),
	}
}

func (p *PlayerData) IsDirty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.dirtyChar ||
		len(p.dirtyItems) > 0 ||
		len(p.dirtySkills) > 0 ||
		len(p.dirtyPets) > 0 ||
		len(p.dirtyQuests) > 0 ||
		len(p.deletedItems) > 0 ||
		len(p.deletedSkills) > 0 ||
		len(p.deletedPets) > 0 ||
		len(p.deletedQuests) > 0
}

func (p *PlayerData) ClearDirty() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clearDirty()
}

func (p *PlayerData) acknowledgeSnapshot(snapshot *DirtySnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.revision == snapshot.revision {
		p.clearDirty()
		return
	}
	p.deletedItems = acknowledgeDeleted(p.deletedItems, snapshot.DeletedItems)
	p.deletedSkills = acknowledgeDeleted(p.deletedSkills, snapshot.DeletedSkills)
	p.deletedPets = acknowledgeDeleted(p.deletedPets, snapshot.DeletedPets)
	p.deletedQuests = acknowledgeDeleted(p.deletedQuests, snapshot.DeletedQuests)
}

func acknowledgeDeleted(current, saved []int64) []int64 {
	if len(current) >= len(saved) && slices.Equal(current[:len(saved)], saved) {
		return current[len(saved):]
	}
	return current
}

func (p *PlayerData) clearDirty() {
	p.dirtyChar = false
	p.dirtyItems = make(map[int64]bool)
	p.dirtySkills = make(map[int64]bool)
	p.dirtyPets = make(map[int64]bool)
	p.dirtyQuests = make(map[int64]bool)
	p.deletedItems = nil
	p.deletedSkills = nil
	p.deletedPets = nil
	p.deletedQuests = nil
}

func (p *PlayerData) SetCharacter(char *character.Character) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Character = cloneCharacter(char)
	p.revision++
	p.dirtyChar = true
}

func (p *PlayerData) GetCharacter() *character.Character {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return cloneCharacter(p.Character)
}

func (p *PlayerData) MarkCharacterDirty() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dirtyChar = true
	p.revision++
}

func (p *PlayerData) SetItem(it *item.Item) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Items[it.ID] = cloneItem(it)
	p.revision++
	p.dirtyItems[it.ID] = true
}

func (p *PlayerData) GetItem(id int64) *item.Item {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return cloneItem(p.Items[id])
}

func (p *PlayerData) GetItems() []*item.Item {
	p.mu.RLock()
	defer p.mu.RUnlock()
	items := make([]*item.Item, 0, len(p.Items))
	for _, it := range p.Items {
		items = append(items, cloneItem(it))
	}
	return items
}

func (p *PlayerData) DeleteItem(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Items, id)
	p.revision++
	delete(p.dirtyItems, id)
	p.deletedItems = append(p.deletedItems, id)
}

func (p *PlayerData) AddItemWithoutDirty(it *item.Item) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Items[it.ID] = cloneItem(it)
	p.revision++
}

func (p *PlayerData) SetSkill(sk *skill.CharacterSkill) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Skills[sk.ID] = cloneSkill(sk)
	p.revision++
	p.dirtySkills[sk.ID] = true
}

func (p *PlayerData) GetSkill(id int64) *skill.CharacterSkill {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return cloneSkill(p.Skills[id])
}

func (p *PlayerData) GetSkills() []*skill.CharacterSkill {
	p.mu.RLock()
	defer p.mu.RUnlock()
	skills := make([]*skill.CharacterSkill, 0, len(p.Skills))
	for _, sk := range p.Skills {
		skills = append(skills, cloneSkill(sk))
	}
	return skills
}

func (p *PlayerData) DeleteSkill(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Skills, id)
	p.revision++
	delete(p.dirtySkills, id)
	p.deletedSkills = append(p.deletedSkills, id)
}

func (p *PlayerData) AddSkillWithoutDirty(sk *skill.CharacterSkill) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Skills[sk.ID] = cloneSkill(sk)
	p.revision++
}

func (p *PlayerData) SetPet(pt *pet.Pet) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Pets[pt.ID] = pt.Clone()
	p.revision++
	p.dirtyPets[pt.ID] = true
}

func (p *PlayerData) GetPet(id int64) *pet.Pet {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Pets[id].Clone()
}

func (p *PlayerData) GetPets() []*pet.Pet {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pets := make([]*pet.Pet, 0, len(p.Pets))
	for _, pt := range p.Pets {
		pets = append(pets, pt.Clone())
	}
	return pets
}

func (p *PlayerData) DeletePet(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Pets, id)
	p.revision++
	delete(p.dirtyPets, id)
	p.deletedPets = append(p.deletedPets, id)
}

func (p *PlayerData) AddPetWithoutDirty(pt *pet.Pet) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Pets[pt.ID] = pt.Clone()
	p.revision++
}

func (p *PlayerData) ApplyPersistedPetFusion(mainPet *pet.Pet, consumedPetIDs []int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if mainPet != nil {
		p.Pets[mainPet.ID] = mainPet.Clone()
		delete(p.dirtyPets, mainPet.ID)
	}
	for _, petID := range consumedPetIDs {
		delete(p.Pets, petID)
		delete(p.dirtyPets, petID)
	}
	p.revision++
}

func (p *PlayerData) SetQuest(q *quest.QuestProgress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Quests[q.ID] = cloneQuest(q)
	p.revision++
	p.dirtyQuests[q.ID] = true
}

func (p *PlayerData) GetQuest(id int64) *quest.QuestProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return cloneQuest(p.Quests[id])
}

func (p *PlayerData) GetQuests() []*quest.QuestProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()
	quests := make([]*quest.QuestProgress, 0, len(p.Quests))
	for _, q := range p.Quests {
		quests = append(quests, cloneQuest(q))
	}
	return quests
}

func (p *PlayerData) DeleteQuest(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Quests, id)
	p.revision++
	delete(p.dirtyQuests, id)
	p.deletedQuests = append(p.deletedQuests, id)
}

func (p *PlayerData) AddQuestWithoutDirty(q *quest.QuestProgress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Quests[q.ID] = cloneQuest(q)
	p.revision++
}

type DirtySnapshot struct {
	revision      uint64
	Character     *character.Character
	DirtyChar     bool
	DirtyItems    map[int64]*item.Item
	DirtySkills   map[int64]*skill.CharacterSkill
	DirtyPets     map[int64]*pet.Pet
	DirtyQuests   map[int64]*quest.QuestProgress
	DeletedItems  []int64
	DeletedSkills []int64
	DeletedPets   []int64
	DeletedQuests []int64
}

func (p *PlayerData) GetDirtySnapshot() *DirtySnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	snapshot := &DirtySnapshot{
		revision:      p.revision,
		Character:     cloneCharacter(p.Character),
		DirtyChar:     p.dirtyChar,
		DirtyItems:    make(map[int64]*item.Item),
		DirtySkills:   make(map[int64]*skill.CharacterSkill),
		DirtyPets:     make(map[int64]*pet.Pet),
		DirtyQuests:   make(map[int64]*quest.QuestProgress),
		DeletedItems:  append([]int64{}, p.deletedItems...),
		DeletedSkills: append([]int64{}, p.deletedSkills...),
		DeletedPets:   append([]int64{}, p.deletedPets...),
		DeletedQuests: append([]int64{}, p.deletedQuests...),
	}

	for id := range p.dirtyItems {
		if it, ok := p.Items[id]; ok {
			snapshot.DirtyItems[id] = cloneItem(it)
		}
	}
	for id := range p.dirtySkills {
		if sk, ok := p.Skills[id]; ok {
			snapshot.DirtySkills[id] = cloneSkill(sk)
		}
	}
	for id := range p.dirtyPets {
		if pt, ok := p.Pets[id]; ok {
			snapshot.DirtyPets[id] = pt.Clone()
		}
	}
	for id := range p.dirtyQuests {
		if q, ok := p.Quests[id]; ok {
			snapshot.DirtyQuests[id] = cloneQuest(q)
		}
	}

	return snapshot
}
