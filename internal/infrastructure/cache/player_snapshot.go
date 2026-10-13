// Detached player snapshots allow serialization and repository I/O without holding cache locks.
package cache

import (
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	"mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/skill"
)

type PlayerSnapshot struct {
	CharacterID int64
	Character   *character.Character
	Items       map[int64]*item.Item
	Skills      map[int64]*skill.CharacterSkill
	Pets        map[int64]*pet.Pet
	Quests      map[int64]*quest.QuestProgress
}

func (p *PlayerData) Snapshot() *PlayerSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := &PlayerSnapshot{
		CharacterID: p.CharacterID,
		Character:   cloneCharacter(p.Character),
		Items:       make(map[int64]*item.Item, len(p.Items)),
		Skills:      make(map[int64]*skill.CharacterSkill, len(p.Skills)),
		Pets:        make(map[int64]*pet.Pet, len(p.Pets)),
		Quests:      make(map[int64]*quest.QuestProgress, len(p.Quests)),
	}
	for id, value := range p.Items {
		result.Items[id] = cloneItem(value)
	}
	for id, value := range p.Skills {
		result.Skills[id] = cloneSkill(value)
	}
	for id, value := range p.Pets {
		result.Pets[id] = value.Clone()
	}
	for id, value := range p.Quests {
		result.Quests[id] = cloneQuest(value)
	}
	return result
}
