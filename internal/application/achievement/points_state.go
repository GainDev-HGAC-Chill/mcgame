// Open-sourced by BaoLT

// Achievement-points state reader derives score milestones from persisted completions.
package achievement

import (
	"context"
	"fmt"

	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/gamedata/models"
)

type AchievementPointsRepository interface {
	GetProgressByCharacter(ctx context.Context, characterID int64) ([]*domainachievement.Progress, error)
}

type AchievementPointsCatalog interface {
	GetAchievement(id int) *models.AchievementTemplate
}

type AchievementPointsStateReader struct {
	repo    AchievementPointsRepository
	catalog AchievementPointsCatalog
}

func NewAchievementPointsStateReader(repo AchievementPointsRepository, catalog AchievementPointsCatalog) *AchievementPointsStateReader {
	return &AchievementPointsStateReader{repo: repo, catalog: catalog}
}

func (r *AchievementPointsStateReader) Name() string {
	return "achievement-points"
}

func (r *AchievementPointsStateReader) RequirementIDs() []int64 {
	return []int64{555, 556, 557, 558, 559}
}

func (r *AchievementPointsStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("achievement points state dependencies not configured")
	}
	rows, err := r.repo.GetProgressByCharacter(ctx, characterID)
	if err != nil {
		return nil, err
	}
	seen := make(map[int64]struct{})
	points := int64(0)
	for _, row := range rows {
		if row == nil || !row.IsCompleted {
			continue
		}
		if _, duplicate := seen[row.AchievementID]; duplicate {
			continue
		}
		template := r.catalog.GetAchievement(int(row.AchievementID))
		if template == nil || template.Enable != 1 || template.Award <= 0 {
			continue
		}
		award := int64(template.Award)
		if float64(award) != template.Award {
			return nil, fmt.Errorf("achievement %d has fractional point award %s", row.AchievementID, catalogNumber(template.Award))
		}
		seen[row.AchievementID] = struct{}{}
		points += award
	}
	progress := boundedStateProgress(points)
	return []StateProof{
		stateProof(555, progress, 1000),
		stateProof(556, progress, 3000),
		stateProof(557, progress, 6000),
		stateProof(558, progress, 12000),
		stateProof(559, progress, 15000),
	}, nil
}
