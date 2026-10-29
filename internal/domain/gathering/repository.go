// Gathering command contracts describe one atomic life-resource completion.
package gathering

import (
	"context"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainskill "mcgame-server/internal/domain/skill"
)

type ResourceKind uint8

const (
	ResourceKindFishing ResourceKind = 1
	ResourceKindHerb    ResourceKind = 2
)

type CompletionCommand struct {
	AttemptID        string
	CharacterID      int64
	Kind             ResourceKind
	NPCID            int
	ResourceLevel    int
	RewardTemplateID int
	ToolItemID       int64
	Succeeded        bool
	VigorCost        int
	MasteryGain      int
}

type CompletionResult struct {
	Character *domainchar.Character
	Skill     *domainskill.CharacterSkill
	Reward    *domainitem.Item
	Tool      *domainitem.Item
	Succeeded bool
}

type CommandRepository interface {
	CompleteGathering(ctx context.Context, command CompletionCommand) (*CompletionResult, error)
}
