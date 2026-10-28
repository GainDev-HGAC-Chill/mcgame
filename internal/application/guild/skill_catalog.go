// Guild skill progression validates catalog authority before any durable command.
package guild

import (
	"context"
	"fmt"
	"math"

	"mcgame-server/internal/gamedata/models"
)

type GuildSkillCommandRepository interface {
	DevelopGuildSkillAtomic(ctx context.Context, guildID, actorID int64, development GuildSkillDevelopment) error
}

type GuildSkillLookup interface {
	GetSkill(id int) *models.SkillTemplate
	GetSkillIDsByUseEnv(useEnv int) []int
}

type GuildSkillDevelopment struct {
	SkillID   int
	Previous  int
	CodeName  string
	Level     int
	ExpCost   int64
	MoneyCost int64
	FamilyIDs []int
}

func ResolveGuildSkillDevelopment(catalog GuildSkillLookup, requestedID int, learnedIDs []int) (GuildSkillDevelopment, error) {
	if catalog == nil || requestedID <= 0 {
		return GuildSkillDevelopment{}, ErrInvalidContribution
	}
	requested := catalog.GetSkill(requestedID)
	if requested == nil || requested.ID != int64(requestedID) || requested.UseEnv != 4 || requested.CodeName == "" || requested.Level < 1 || requested.Level > 6 || requested.Level != math.Trunc(requested.Level) || requested.CostGuildContrib != 0 {
		return GuildSkillDevelopment{}, ErrUnsupportedAction
	}
	exp, ok := guildSkillCost(requested.GuildDevExp)
	if !ok {
		return GuildSkillDevelopment{}, ErrUnsupportedAction
	}
	money, ok := guildSkillCost(requested.GuildDevMoney)
	if !ok {
		return GuildSkillDevelopment{}, ErrUnsupportedAction
	}
	familyIDs := make([]int, 0, 6)
	levelIDs := make(map[int]int)
	for _, id := range catalog.GetSkillIDsByUseEnv(4) {
		row := catalog.GetSkill(id)
		if row == nil || row.ID != int64(id) || row.UseEnv != 4 {
			return GuildSkillDevelopment{}, fmt.Errorf("invalid guild skill catalog row %d", id)
		}
		if row.CodeName != requested.CodeName {
			continue
		}
		if row.Level < 1 || row.Level > 6 || row.Level != math.Trunc(row.Level) || levelIDs[int(row.Level)] != 0 {
			return GuildSkillDevelopment{}, fmt.Errorf("ambiguous guild skill catalog family %q", requested.CodeName)
		}
		levelIDs[int(row.Level)] = id
		familyIDs = append(familyIDs, id)
	}
	if len(familyIDs) != 6 || levelIDs[int(requested.Level)] != requestedID {
		return GuildSkillDevelopment{}, fmt.Errorf("incomplete guild skill catalog family %q", requested.CodeName)
	}
	currentID, currentLevel := 0, 0
	for _, id := range learnedIDs {
		learned := catalog.GetSkill(id)
		if learned == nil || learned.UseEnv != 4 || learned.ID != int64(id) {
			return GuildSkillDevelopment{}, fmt.Errorf("invalid persisted guild skill %d", id)
		}
		if learned.CodeName != requested.CodeName {
			continue
		}
		if currentID != 0 || learned.Level < 1 || learned.Level > 6 || learned.Level != math.Trunc(learned.Level) {
			return GuildSkillDevelopment{}, fmt.Errorf("ambiguous guild skill family %q", requested.CodeName)
		}
		currentID, currentLevel = id, int(learned.Level)
	}
	if int(requested.Level) != currentLevel+1 {
		return GuildSkillDevelopment{}, ErrUnsupportedAction
	}
	return GuildSkillDevelopment{SkillID: requestedID, Previous: currentID, CodeName: requested.CodeName, Level: int(requested.Level), ExpCost: exp, MoneyCost: money, FamilyIDs: familyIDs}, nil
}

func guildSkillCost(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= float64(math.MaxInt64) || value != math.Trunc(value) {
		return 0, false
	}
	return int64(value), true
}
