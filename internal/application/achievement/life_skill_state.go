// Open-sourced by BaoLT

// Life-skill state reader proves learned professions and recipe ownership from persisted skills.
package achievement

import (
	"context"
	"fmt"

	appskill "mcgame-server/internal/application/skill"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata/models"
)

type LifeSkillStateRepository interface {
	FindByCharacterID(ctx context.Context, characterID int64) ([]*domainskill.CharacterSkill, error)
}

type LifeSkillCatalog interface {
	GetSkill(id int) *models.SkillTemplate
}

type LifeSkillStateReader struct {
	repo    LifeSkillStateRepository
	catalog LifeSkillCatalog
}

var medicineRecipeRequirements = map[int64]int{
	281: 5048,
	282: 5050,
	283: 5054,
	284: 5056,
	285: 5059,
	286: 5220,
}

func NewLifeSkillStateReader(repo LifeSkillStateRepository, catalog LifeSkillCatalog) *LifeSkillStateReader {
	return &LifeSkillStateReader{repo: repo, catalog: catalog}
}

func (r *LifeSkillStateReader) Name() string {
	return "life-skill"
}

func (r *LifeSkillStateReader) RequirementIDs() []int64 {
	return []int64{276, 281, 282, 283, 284, 285, 286, 295, 296, 297, 298, 299, 300, 301}
}

func (r *LifeSkillStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("life-skill state dependencies not configured")
	}
	rows, err := r.repo.FindByCharacterID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	owned := make(map[int]struct{}, len(rows))
	lifeLevels := make(map[int]int, 5)
	cookbookCount := 0
	for _, row := range rows {
		if row == nil || row.CharacterID != characterID || row.SkillID <= 0 {
			continue
		}
		template := r.catalog.GetSkill(row.SkillID)
		if template == nil || template.ID != int64(row.SkillID) {
			continue
		}
		_, duplicate := owned[row.SkillID]
		owned[row.SkillID] = struct{}{}
		skillType := int(template.Type)
		if float64(skillType) != template.Type {
			continue
		}
		if skillType == appskill.CookbookSkillType && !duplicate {
			cookbookCount++
		}
		if isAchievementLifeSkillType(skillType) && row.Level > lifeLevels[skillType] {
			lifeLevels[skillType] = row.Level
		}
	}

	proofs := []StateProof{stateProof(276, cookbookCount, 35)}
	for _, requirementID := range []int64{281, 282, 283, 284, 285, 286} {
		_, learned := owned[medicineRecipeRequirements[requirementID]]
		progress := 0
		if learned {
			progress = 1
		}
		proofs = append(proofs, stateProof(requirementID, progress, 1))
	}
	learnedLifeSkills := 0
	maxLifeSkillLevel := 0
	for _, skillType := range achievementLifeSkillTypes() {
		level := lifeLevels[skillType]
		if level > 0 {
			learnedLifeSkills++
		}
		if level > maxLifeSkillLevel {
			maxLifeSkillLevel = level
		}
	}
	proofs = append(proofs,
		stateProof(295, learnedLifeSkills, 1),
		stateProof(296, lifeLevels[appskill.PlantSkillType], 1),
		stateProof(297, lifeLevels[appskill.HerbSkillType], 1),
		stateProof(298, lifeLevels[appskill.FishSkillType], 1),
		stateProof(299, lifeLevels[appskill.MedicineSkillType], 1),
		stateProof(300, lifeLevels[appskill.CookSkillType], 1),
		stateProof(301, maxLifeSkillLevel, 6),
	)
	return proofs, nil
}

func achievementLifeSkillTypes() []int {
	return []int{appskill.FishSkillType, appskill.PlantSkillType, appskill.HerbSkillType, appskill.CookSkillType, appskill.MedicineSkillType}
}

func isAchievementLifeSkillType(skillType int) bool {
	for _, candidate := range achievementLifeSkillTypes() {
		if skillType == candidate {
			return true
		}
	}
	return false
}
