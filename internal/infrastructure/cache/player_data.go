// Open-sourced by BaoLT

// PlayerData holds all cached entities for a single player/character.
// Provides dirty tracking for write-behind persistence.
// Thread-safe via sync.RWMutex for concurrent RPC access.
package cache

import (
	"slices"
	"sync"

	domainactivity "mcgame-server/internal/domain/activity"
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

	mu                sync.RWMutex
	mutationMu        sync.Mutex
	persistMu         sync.Mutex
	redisMu           sync.Mutex
	revision          uint64
	characterRevision uint64
	itemRevisions     map[int64]uint64
	petRevisions      map[int64]uint64
}

func NewPlayerData(charID int64) *PlayerData {
	return &PlayerData{
		CharacterID:       charID,
		Items:             make(map[int64]*item.Item),
		Skills:            make(map[int64]*skill.CharacterSkill),
		Pets:              make(map[int64]*pet.Pet),
		Quests:            make(map[int64]*quest.QuestProgress),
		dirtyItems:        make(map[int64]bool),
		dirtySkills:       make(map[int64]bool),
		dirtyPets:         make(map[int64]bool),
		dirtyQuests:       make(map[int64]bool),
		characterRevision: 1,
		itemRevisions:     make(map[int64]uint64),
		petRevisions:      make(map[int64]uint64),
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
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
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
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.setCharacter(char)
}

func (p *PlayerData) SetCharacterAtRevision(char *character.Character) bool {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	if char == nil || char.CacheRevision != p.characterRevision {
		return false
	}
	p.setCharacter(char)
	return true
}

func (p *PlayerData) setCharacter(char *character.Character) {
	p.Character = cloneCharacter(char)
	p.characterRevision++
	p.Character.CacheRevision = p.characterRevision
	p.revision++
	p.dirtyChar = true
}

func (p *PlayerData) GetCharacter() *character.Character {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := cloneCharacter(p.Character)
	if result != nil {
		result.CacheRevision = p.characterRevision
	}
	return result
}

func (p *PlayerData) SetCharacterGoldWithoutDirty(gold int64) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Character == nil {
		return
	}
	p.Character.Gold = gold
	p.characterRevision++
	p.revision++
}

func (p *PlayerData) SetCharacterNameWithoutDirty(name string) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.setCharacterNameWithoutDirty(name)
}

func (p *PlayerData) setCharacterNameWithoutDirty(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Character == nil {
		return
	}
	p.Character.Name = name
	p.characterRevision++
	p.revision++
}

func (p *PlayerData) SyncCommittedGuildState(guildID *int64, money, moneyBind, gold, guildContrib, donateContrib, restoreNormal, restoreDonate int64) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.syncCommittedGuildState(guildID, money, moneyBind, gold, guildContrib, donateContrib, restoreNormal, restoreDonate)
}

func (p *PlayerData) syncCommittedGuildState(guildID *int64, money, moneyBind, gold, guildContrib, donateContrib, restoreNormal, restoreDonate int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Character == nil {
		return
	}
	p.Character.GuildID = clonePointer(guildID)
	p.Character.Money = money
	p.Character.MoneyBind = moneyBind
	p.Character.Gold = gold
	p.Character.GuildContrib = int(guildContrib)
	p.Character.DonateContrib = int(donateContrib)
	p.Character.GuildRestoreContrib = int(restoreNormal)
	p.Character.GuildRestoreDonate = int(restoreDonate)
	p.characterRevision++
	p.revision++
}

func (p *PlayerData) MarkCharacterDirty() {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dirtyChar = true
	p.characterRevision++
	p.revision++
}

func (p *PlayerData) SetItem(it *item.Item) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.setItem(it)
}

func (p *PlayerData) SetItemAtRevision(it *item.Item) bool {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	if it == nil || it.CacheRevision != p.itemRevision(it.ID) {
		return false
	}
	p.setItem(it)
	return true
}

func (p *PlayerData) setItem(it *item.Item) {
	p.Items[it.ID] = cloneItem(it)
	p.itemRevisions[it.ID] = p.itemRevision(it.ID) + 1
	p.Items[it.ID].CacheRevision = p.itemRevisions[it.ID]
	p.revision++
	p.dirtyItems[it.ID] = true
}

func (p *PlayerData) GetItem(id int64) *item.Item {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := cloneItem(p.Items[id])
	if result != nil {
		result.CacheRevision = p.itemRevision(id)
	}
	return result
}

func (p *PlayerData) GetItems() []*item.Item {
	p.mu.RLock()
	defer p.mu.RUnlock()
	items := make([]*item.Item, 0, len(p.Items))
	for _, it := range p.Items {
		cloned := cloneItem(it)
		cloned.CacheRevision = p.itemRevision(it.ID)
		items = append(items, cloned)
	}
	return items
}

func (p *PlayerData) DeleteItem(id int64) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.deleteItem(id)
}

func (p *PlayerData) DeleteItemAtRevision(id int64, expected uint64) bool {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	if expected != p.itemRevision(id) {
		return false
	}
	p.deleteItem(id)
	return true
}

func (p *PlayerData) deleteItem(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Items, id)
	p.itemRevisions[id] = p.itemRevision(id) + 1
	p.revision++
	delete(p.dirtyItems, id)
	p.deletedItems = append(p.deletedItems, id)
}

func (p *PlayerData) AddItemWithoutDirty(it *item.Item) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Items[it.ID] = cloneItem(it)
	p.itemRevisions[it.ID] = p.itemRevision(it.ID) + 1
	p.Items[it.ID].CacheRevision = p.itemRevisions[it.ID]
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
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.setPet(pt)
}

func (p *PlayerData) SetPetAtRevision(pt *pet.Pet) bool {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	if pt == nil || pt.CacheRevision != p.petRevision(pt.ID) {
		return false
	}
	p.setPetLocked(pt)
	return true
}

func (p *PlayerData) setPet(pt *pet.Pet) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.setPetLocked(pt)
}

func (p *PlayerData) setPetLocked(pt *pet.Pet) {
	p.Pets[pt.ID] = pt.Clone()
	p.petRevisions[pt.ID] = p.petRevision(pt.ID) + 1
	p.Pets[pt.ID].CacheRevision = p.petRevisions[pt.ID]
	p.revision++
	p.dirtyPets[pt.ID] = true
}

func (p *PlayerData) GetPet(id int64) *pet.Pet {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := p.Pets[id].Clone()
	if result != nil {
		result.CacheRevision = p.petRevision(id)
	}
	return result
}

func (p *PlayerData) GetPets() []*pet.Pet {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pets := make([]*pet.Pet, 0, len(p.Pets))
	for _, pt := range p.Pets {
		cloned := pt.Clone()
		cloned.CacheRevision = p.petRevision(pt.ID)
		pets = append(pets, cloned)
	}
	return pets
}

func (p *PlayerData) DeletePet(id int64) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.deletePet(id)
}

func (p *PlayerData) DeletePetAtRevision(id int64, expected uint64) bool {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	if expected != p.petRevision(id) {
		return false
	}
	p.deletePet(id)
	return true
}

func (p *PlayerData) deletePet(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Pets, id)
	p.petRevisions[id] = p.petRevision(id) + 1
	p.revision++
	delete(p.dirtyPets, id)
	p.deletedPets = append(p.deletedPets, id)
}

func (p *PlayerData) AddPetWithoutDirty(pt *pet.Pet) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Pets[pt.ID] = pt.Clone()
	p.petRevisions[pt.ID] = p.petRevision(pt.ID) + 1
	p.Pets[pt.ID].CacheRevision = p.petRevisions[pt.ID]
	p.revision++
}

func (p *PlayerData) ApplyPersistedPetFusion(mainPet *pet.Pet, consumedPetIDs []int64) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.applyPersistedPetFusion(mainPet, consumedPetIDs)
}

func (p *PlayerData) applyPersistedPetFusion(mainPet *pet.Pet, consumedPetIDs []int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if mainPet != nil {
		p.Pets[mainPet.ID] = mainPet.Clone()
		p.petRevisions[mainPet.ID] = p.petRevision(mainPet.ID) + 1
		p.Pets[mainPet.ID].CacheRevision = p.petRevisions[mainPet.ID]
		delete(p.dirtyPets, mainPet.ID)
	}
	for _, petID := range consumedPetIDs {
		delete(p.Pets, petID)
		p.petRevisions[petID] = p.petRevision(petID) + 1
		delete(p.dirtyPets, petID)
	}
	p.revision++
}

func (p *PlayerData) ApplyPersistedPetIllusion(mainPet *pet.Pet, consumedPetID, luckItemID int64, remainingLuck int, updateMain bool) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.applyPersistedPetIllusion(mainPet, consumedPetID, luckItemID, remainingLuck, updateMain)
}

func (p *PlayerData) applyPersistedPetIllusion(mainPet *pet.Pet, consumedPetID, luckItemID int64, remainingLuck int, updateMain bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if updateMain {
		p.Pets[mainPet.ID] = mainPet.Clone()
		p.petRevisions[mainPet.ID] = p.petRevision(mainPet.ID) + 1
		p.Pets[mainPet.ID].CacheRevision = p.petRevisions[mainPet.ID]
		delete(p.dirtyPets, mainPet.ID)
	}
	delete(p.Pets, consumedPetID)
	p.petRevisions[consumedPetID] = p.petRevision(consumedPetID) + 1
	delete(p.dirtyPets, consumedPetID)
	if luckItemID > 0 {
		if remainingLuck == 0 {
			delete(p.Items, luckItemID)
		} else if current := p.Items[luckItemID]; current != nil {
			updated := cloneItem(current)
			updated.StackCount = remainingLuck
			p.Items[luckItemID] = updated
		}
		delete(p.dirtyItems, luckItemID)
		p.itemRevisions[luckItemID] = p.itemRevision(luckItemID) + 1
	}
	p.revision++
}

func (p *PlayerData) ApplyPersistedPetSkillBook(updatedPet *pet.Pet, bookItemID int64, remainingStack int) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.applyPersistedPetSkillBook(updatedPet, bookItemID, remainingStack)
}

func (p *PlayerData) applyPersistedPetSkillBook(updatedPet *pet.Pet, bookItemID int64, remainingStack int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Pets[updatedPet.ID] = updatedPet.Clone()
	p.petRevisions[updatedPet.ID] = p.petRevision(updatedPet.ID) + 1
	p.Pets[updatedPet.ID].CacheRevision = p.petRevisions[updatedPet.ID]
	delete(p.dirtyPets, updatedPet.ID)
	if remainingStack == 0 {
		delete(p.Items, bookItemID)
	} else if current := p.Items[bookItemID]; current != nil {
		updated := cloneItem(current)
		updated.StackCount = remainingStack
		p.Items[bookItemID] = updated
	}
	delete(p.dirtyItems, bookItemID)
	p.itemRevisions[bookItemID] = p.itemRevision(bookItemID) + 1
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

// SyncCommittedItemCount cập nhật cache sau khi PostgreSQL
// đã commit việc trừ vật phẩm. Không đánh dấu dirty.
func (p *PlayerData) SyncCommittedItemCount(
	itemID int64,
	remaining int,
) {
	p.mutationMu.Lock()
	defer p.mutationMu.Unlock()
	p.syncCommittedItemCount(itemID, remaining)
}

func (p *PlayerData) syncCommittedItemCount(itemID int64, remaining int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current, exists := p.Items[itemID]
	if !exists {
		return
	}

	if remaining <= 0 {
		delete(p.Items, itemID)
	} else {
		updated := cloneItem(current)
		updated.StackCount = remaining
		p.Items[itemID] = updated
	}

	delete(p.dirtyItems, itemID)
	p.itemRevisions[itemID] = p.itemRevision(itemID) + 1

	p.deletedItems = slices.DeleteFunc(
		p.deletedItems,
		func(id int64) bool {
			return id == itemID
		},
	)

	p.revision++
}

func (p *PlayerData) syncCommittedSendCombine(result *domainactivity.SendCombinePurchaseResult) {
	if result == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Character != nil {
		p.Character.Gold = result.Gold
		p.dirtyChar = false
	}
	for _, committed := range result.Items {
		if committed == nil {
			continue
		}
		p.Items[committed.ID] = cloneItem(committed)
		delete(p.dirtyItems, committed.ID)
		p.itemRevisions[committed.ID] = p.itemRevision(committed.ID) + 1
		p.deletedItems = slices.DeleteFunc(p.deletedItems, func(id int64) bool { return id == committed.ID })
	}
	p.revision++
}

func (p *PlayerData) syncCommittedGuildLoop(progressID int64, result *quest.GuildLoopCompletionResult) {
	if result == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Character != nil {
		p.Character.GuildID = clonePointer(&result.GuildID)
		p.Character.Money = result.Money
		p.Character.MoneyBind = result.MoneyBind
		p.Character.Gold = result.Gold
		p.Character.GuildContrib = int(result.GuildContrib)
		p.Character.DonateContrib = int(result.DonateContrib)
		p.Character.GuildRestoreContrib = int(result.GuildRestoreNormal)
		p.Character.GuildRestoreDonate = int(result.GuildRestoreDonate)
		p.characterRevision++
	}
	for _, state := range result.ItemStates {
		if state.Remaining <= 0 {
			delete(p.Items, state.ItemID)
		} else if current := p.Items[state.ItemID]; current != nil {
			updated := cloneItem(current)
			updated.StackCount = state.Remaining
			p.Items[state.ItemID] = updated
		}
		delete(p.dirtyItems, state.ItemID)
		p.itemRevisions[state.ItemID] = p.itemRevision(state.ItemID) + 1
		p.deletedItems = slices.DeleteFunc(p.deletedItems, func(id int64) bool { return id == state.ItemID })
	}
	for _, petID := range result.ConsumedPetIDs {
		delete(p.Pets, petID)
		delete(p.dirtyPets, petID)
		p.petRevisions[petID] = p.petRevision(petID) + 1
		p.deletedPets = slices.DeleteFunc(p.deletedPets, func(id int64) bool { return id == petID })
	}
	delete(p.Quests, progressID)
	delete(p.dirtyQuests, progressID)
	p.deletedQuests = slices.DeleteFunc(p.deletedQuests, func(id int64) bool { return id == progressID })
	if result.NextQuest != nil {
		p.Quests[result.NextQuest.ID] = cloneQuest(result.NextQuest)
		delete(p.dirtyQuests, result.NextQuest.ID)
	}
	p.revision++
}

func (p *PlayerData) itemRevision(id int64) uint64 {
	if revision := p.itemRevisions[id]; revision > 0 {
		return revision
	}
	return 1
}

func (p *PlayerData) petRevision(id int64) uint64 {
	if revision := p.petRevisions[id]; revision > 0 {
		return revision
	}
	return 1
}
