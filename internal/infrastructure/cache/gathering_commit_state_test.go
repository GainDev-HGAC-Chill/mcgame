package cache

import (
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domaingathering "mcgame-server/internal/domain/gathering"
	domainitem "mcgame-server/internal/domain/item"
	domainskill "mcgame-server/internal/domain/skill"
)

func TestSyncCommittedGatheringReconcilesWithoutDirtyWrites(t *testing.T) {
	data := NewPlayerData(3)
	data.Character = &domainchar.Character{ID: 3, CurrentSP: 20, FishDex: 10}
	data.Skills[50] = &domainskill.CharacterSkill{ID: 50, CharacterID: 3, SkillID: 2400, Level: 1, Exp: 10}
	data.Items[70] = &domainitem.Item{ID: 70, CharacterID: 3, TemplateID: 1437, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 13}
	data.dirtyChar = true
	data.dirtySkills[50] = true
	data.dirtyItems[70] = true
	data.dirtyItems[80] = true
	data.deletedItems = []int64{80}

	result := &domaingathering.CompletionResult{
		Character: &domainchar.Character{ID: 3, CurrentSP: 18, FishDex: 12},
		Skill:     &domainskill.CharacterSkill{ID: 50, CharacterID: 3, SkillID: 2400, Level: 1, Exp: 12},
		Reward:    &domainitem.Item{ID: 80, CharacterID: 3, TemplateID: 2361, SlotType: domainitem.SlotTypeBag, StackCount: 1},
		Tool:      &domainitem.Item{ID: 70, CharacterID: 3, TemplateID: 1437, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 13},
		Succeeded: true,
	}
	data.syncCommittedGathering(result)

	if data.Character.CurrentSP != 18 || data.Character.FishDex != 12 || data.Skills[50].Exp != 12 || data.Items[80] == nil {
		t.Fatalf("committed gathering state not reconciled: character=%+v skill=%+v items=%+v", data.Character, data.Skills[50], data.Items)
	}
	if data.IsDirty() {
		t.Fatalf("committed gathering state remained dirty: %+v", data.GetDirtySnapshot())
	}
}
