// PostgreSQL guild-loop commands return the committed player, guild, inventory, and next-round snapshot.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainquest "mcgame-server/internal/domain/quest"
)

func (r *QuestLoopRepository) CompleteGuildLoopAtomic(ctx context.Context, command domainquest.GuildLoopCompletionCommand) (*domainquest.GuildLoopCompletionResult, error) {
	items := command.Items
	if items == nil {
		items = make([]domainquest.GuildLoopItemConsumption, 0)
	}
	petIDs := command.PetIDs
	if petIDs == nil {
		petIDs = make([]int64, 0)
	}
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("encode guild loop item consumption: %w", err)
	}
	var nextQuestID *int
	var nextObjectives []byte
	var nextStartedAt *time.Time
	if command.NextQuest != nil {
		nextQuestID = &command.NextQuest.QuestID
		nextObjectives, err = json.Marshal(command.NextQuest.Objectives)
		if err != nil {
			return nil, fmt.Errorf("encode guild loop next objectives: %w", err)
		}
		nextStartedAt = &command.NextQuest.StartedAt
	}
	result := &domainquest.GuildLoopCompletionResult{}
	var itemStatesJSON []byte
	var consumedPetIDs []int64
	var nextProgressID *int64
	var committedNextQuestID *int
	var committedNextObjectives []byte
	var committedNextStartedAt *time.Time
	err = r.db.pool.QueryRow(ctx, "select * from player.complete_guild_loop_atomic($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12::jsonb,$13)",
		command.CharacterID, command.ProgressID, command.QuestID, command.ExpectedFt,
		command.Money, command.MoneyBind, command.Contribution, command.GuildExp,
		itemsJSON, petIDs, nextQuestID, nextObjectives, nextStartedAt,
	).Scan(
		&result.GuildID, &result.Money, &result.MoneyBind, &result.Gold,
		&result.GuildContrib, &result.DonateContrib, &result.GuildRestoreNormal, &result.GuildRestoreDonate,
		&result.MemberNormal, &result.MemberTotal, &result.MemberWeekly, &result.GuildExperience,
		&result.Ft, &result.Active, &itemStatesJSON, &consumedPetIDs,
		&nextProgressID, &committedNextQuestID, &committedNextObjectives, &committedNextStartedAt,
		&result.DiaryTaskCount, &result.DiaryVitality, &result.DiaryVitalityDelta,
	)
	if err != nil {
		return nil, fmt.Errorf("complete guild loop atomically: %w", err)
	}
	if err := json.Unmarshal(itemStatesJSON, &result.ItemStates); err != nil {
		return nil, fmt.Errorf("decode guild loop item states: %w", err)
	}
	result.ConsumedPetIDs = consumedPetIDs
	if nextProgressID != nil && committedNextQuestID != nil && committedNextStartedAt != nil {
		next := &domainquest.QuestProgress{ID: *nextProgressID, CharacterID: command.CharacterID, QuestID: *committedNextQuestID, Status: domainquest.QuestStatusActive, StartedAt: *committedNextStartedAt}
		if err := json.Unmarshal(committedNextObjectives, &next.Objectives); err != nil {
			return nil, fmt.Errorf("decode committed guild loop next objectives: %w", err)
		}
		result.NextQuest = next
	}
	return result, nil
}

var _ domainquest.GuildLoopCommandRepository = (*QuestLoopRepository)(nil)
