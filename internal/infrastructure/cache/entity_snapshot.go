// Cache entities are detached on ingress, reads and persistence snapshots.
// Callers mutate their own copy and explicitly save it through the repository.
package cache

import (
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/skill"
	"mcgame-server/pkg/valuecopy"
)

func clonePointer[T any](source *T) *T {
	if source == nil {
		return nil
	}
	result := *source
	return &result
}

func cloneCharacter(source *character.Character) *character.Character {
	result := clonePointer(source)
	if result != nil {
		result.GuildID = clonePointer(source.GuildID)
		result.VIPExpiresAt = clonePointer(source.VIPExpiresAt)
		result.PMProcessData = valuecopy.Copy(source.PMProcessData)
		result.PetGuardData = valuecopy.Copy(source.PetGuardData)
		result.BossDaily = valuecopy.Copy(source.BossDaily)
		result.GuideLog = valuecopy.Copy(source.GuideLog)
	}
	return result
}

func cloneItem(source *item.Item) *item.Item {
	result := clonePointer(source)
	if result != nil {
		result.Durability = clonePointer(source.Durability)
		result.MaxDurability = clonePointer(source.MaxDurability)
		result.Properties = valuecopy.Copy(source.Properties)
	}
	return result
}

func cloneSkill(source *skill.CharacterSkill) *skill.CharacterSkill {
	result := clonePointer(source)
	if result != nil {
		result.SlotPosition = clonePointer(source.SlotPosition)
		result.CooldownEnd = clonePointer(source.CooldownEnd)
	}
	return result
}

func cloneQuest(source *quest.QuestProgress) *quest.QuestProgress {
	result := clonePointer(source)
	if result != nil {
		result.Objectives = valuecopy.Copy(source.Objectives)
		result.CompletedAt = clonePointer(source.CompletedAt)
		result.ExpiresAt = clonePointer(source.ExpiresAt)
		result.LastReset = clonePointer(source.LastReset)
	}
	return result
}
