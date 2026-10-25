// Open-sourced by BaoLT

// Achievement progress evaluators cover only requirement families with verified authority.
package achievement

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	domaincombat "mcgame-server/internal/domain/combat"

	"github.com/google/uuid"
)

type requirementTarget struct {
	RequirementID int64
	AchievementID int64
	Target        int
}

var levelRequirements = []requirementTarget{
	{RequirementID: 1, AchievementID: 1001, Target: 10},
	{RequirementID: 2, AchievementID: 1002, Target: 50},
	{RequirementID: 3, AchievementID: 1003, Target: 100},
	{RequirementID: 4, AchievementID: 1004, Target: 140},
}

var loginRequirements = []requirementTarget{
	{RequirementID: 53, AchievementID: 1051, Target: 8},
	{RequirementID: 54, AchievementID: 1052, Target: 28},
	{RequirementID: 55, AchievementID: 1053, Target: 88},
	{RequirementID: 56, AchievementID: 1054, Target: 218},
	{RequirementID: 423, AchievementID: 6098, Target: 588},
	{RequirementID: 424, AchievementID: 6099, Target: 988},
}

var questRequirements = []requirementTarget{
	{RequirementID: 248, AchievementID: 5042, Target: 50},
	{RequirementID: 249, AchievementID: 5043, Target: 500},
	{RequirementID: 250, AchievementID: 5044, Target: 1000},
	{RequirementID: 251, AchievementID: 5045, Target: 5000},
}

var battleRequirements = []requirementTarget{
	{RequirementID: 185, AchievementID: 4035, Target: 1},
	{RequirementID: 186, AchievementID: 4036, Target: 10},
	{RequirementID: 187, AchievementID: 4037, Target: 100},
	{RequirementID: 188, AchievementID: 4038, Target: 1000},
	{RequirementID: 189, AchievementID: 4039, Target: 10000},
}

var arenaRequirements = []requirementTarget{
	{RequirementID: 151, AchievementID: 4001, Target: 5},
	{RequirementID: 152, AchievementID: 4002, Target: 50},
	{RequirementID: 153, AchievementID: 4003, Target: 500},
	{RequirementID: 154, AchievementID: 4004, Target: 1000},
}

func (s *Service) RecordLogin(ctx context.Context, charID int64, level int, now time.Time) (domainachievement.ProgressUpdate, error) {
	if charID <= 0 || level <= 0 || now.IsZero() {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement login event")
	}
	levelResult, err := s.RecordLevel(ctx, charID, level, now)
	if err != nil {
		return domainachievement.ProgressUpdate{}, err
	}
	loginResult, err := s.recordLoginDay(ctx, charID, loginRequirements, now)
	if err != nil {
		return domainachievement.ProgressUpdate{}, err
	}
	return mergeProgressUpdates(levelResult, loginResult), nil
}

func (s *Service) RecordLevel(ctx context.Context, charID int64, level int, now time.Time) (domainachievement.ProgressUpdate, error) {
	if charID <= 0 || level <= 0 || now.IsZero() {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement level event")
	}
	updates := make([]domainachievement.RequirementUpdate, 0, len(levelRequirements))
	for _, requirement := range levelRequirements {
		absolute := level
		updates = append(updates, domainachievement.RequirementUpdate{
			RequirementID: requirement.RequirementID,
			AchievementID: requirement.AchievementID,
			Absolute:      &absolute,
			Target:        requirement.Target,
		})
	}
	return s.recordEvent(ctx, charID, fmt.Sprintf("level:v1:%d", level), updates, now)
}

func (s *Service) RecordQuestFinished(ctx context.Context, charID int64, questID int, progressID int64, completionCount int, now time.Time) (domainachievement.ProgressUpdate, error) {
	if charID <= 0 || questID <= 0 || progressID <= 0 || completionCount <= 0 || now.IsZero() {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement quest event")
	}
	eventKey := fmt.Sprintf("quest:v1:%d:%d:%d", questID, progressID, completionCount)
	return s.recordIncrement(ctx, charID, eventKey, questRequirements, now)
}

func (s *Service) RecordBattleEnded(ctx context.Context, charID int64, battleID string, battleType domaincombat.BattleType, won bool, now time.Time) (domainachievement.ProgressUpdate, error) {
	parsedBattleID, parseErr := uuid.Parse(battleID)
	if charID <= 0 || battleID == "" || strings.TrimSpace(battleID) != battleID || parseErr != nil || parsedBattleID.String() != battleID || now.IsZero() || !supportedAchievementBattleType(battleType) {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement battle event")
	}
	if !won {
		return emptyProgressUpdate(), nil
	}
	requirements := append([]requirementTarget{}, battleRequirements...)
	if battleType == domaincombat.BattleTypeArena {
		requirements = append(requirements, arenaRequirements...)
	}
	return s.recordIncrement(ctx, charID, "battle:v1:"+battleID, requirements, now)
}

func supportedAchievementBattleType(battleType domaincombat.BattleType) bool {
	switch battleType {
	case domaincombat.BattleTypePVE,
		domaincombat.BattleTypePVP,
		domaincombat.BattleTypeArena,
		domaincombat.BattleTypeBoss,
		domaincombat.BattleTypeStarInstance:
		return true
	default:
		return false
	}
}

func (s *Service) recordIncrement(ctx context.Context, charID int64, eventKey string, requirements []requirementTarget, now time.Time) (domainachievement.ProgressUpdate, error) {
	updates := buildIncrementUpdates(requirements)
	return s.recordEvent(ctx, charID, eventKey, updates, now)
}

func (s *Service) recordLoginDay(ctx context.Context, charID int64, requirements []requirementTarget, now time.Time) (domainachievement.ProgressUpdate, error) {
	if s.repo == nil {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("achievement repository not configured")
	}
	if charID <= 0 || len(requirements) == 0 || now.IsZero() {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement login day")
	}
	return s.repo.RecordLoginDay(ctx, charID, buildIncrementUpdates(requirements), now.UTC())
}

func buildIncrementUpdates(requirements []requirementTarget) []domainachievement.RequirementUpdate {
	updates := make([]domainachievement.RequirementUpdate, 0, len(requirements))
	for _, requirement := range requirements {
		updates = append(updates, domainachievement.RequirementUpdate{
			RequirementID: requirement.RequirementID,
			AchievementID: requirement.AchievementID,
			Delta:         1,
			Target:        requirement.Target,
		})
	}
	return updates
}

func (s *Service) recordEvent(ctx context.Context, charID int64, eventKey string, updates []domainachievement.RequirementUpdate, now time.Time) (domainachievement.ProgressUpdate, error) {
	if s.repo == nil {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("achievement repository not configured")
	}
	if charID <= 0 || eventKey == "" || len(updates) == 0 {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement event")
	}
	return s.repo.RecordEvent(ctx, charID, eventKey, updates, now.UTC())
}

func mergeProgressUpdates(results ...domainachievement.ProgressUpdate) domainachievement.ProgressUpdate {
	merged := emptyProgressUpdate()
	seenCompletion := map[int64]bool{}
	for _, result := range results {
		merged.Requirements = append(merged.Requirements, result.Requirements...)
		for _, completion := range result.Completed {
			if seenCompletion[completion.AchievementID] {
				continue
			}
			seenCompletion[completion.AchievementID] = true
			merged.Completed = append(merged.Completed, completion)
		}
	}
	return merged
}

func emptyProgressUpdate() domainachievement.ProgressUpdate {
	return domainachievement.ProgressUpdate{
		Requirements: []*domainachievement.RequirementProgress{},
		Completed:    []domainachievement.CompletionEntry{},
	}
}
