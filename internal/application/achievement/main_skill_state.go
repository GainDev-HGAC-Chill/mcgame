// Open-sourced by BaoLT

// Main-skill state reader proves current-class skill-family level milestones.
package achievement

import (
	"context"
	"fmt"

	domaincharacter "mcgame-server/internal/domain/character"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata/models"
)

const mainSkillFamilyCount = 10

type MainSkillCharacterRepository interface {
	FindByID(ctx context.Context, id int64) (*domaincharacter.Character, error)
}

type MainSkillRepository interface {
	FindByCharacterID(ctx context.Context, characterID int64) ([]*domainskill.CharacterSkill, error)
}

type MainSkillCatalog interface {
	GetSkill(id int) *models.SkillTemplate
	GetAllSkills() []*models.SkillTemplate
}

type MainSkillStateReader struct {
	characterRepo MainSkillCharacterRepository
	skillRepo     MainSkillRepository
	catalog       MainSkillCatalog
}

func NewMainSkillStateReader(characterRepo MainSkillCharacterRepository, skillRepo MainSkillRepository, catalog MainSkillCatalog) *MainSkillStateReader {
	return &MainSkillStateReader{characterRepo: characterRepo, skillRepo: skillRepo, catalog: catalog}
}

func (r *MainSkillStateReader) Name() string {
	return "main-skill"
}

func (r *MainSkillStateReader) RequirementIDs() []int64 {
	return requirementIDRange(5, 8)
}

func (r *MainSkillStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.characterRepo == nil || r.skillRepo == nil || r.catalog == nil {
		return nil, fmt.Errorf("main-skill state dependencies not configured")
	}
	character, err := r.characterRepo.FindByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if character == nil || character.ID != characterID {
		return nil, fmt.Errorf("character %d not found", characterID)
	}
	families, err := mainSkillFamilies(r.catalog.GetAllSkills(), character.ClassID)
	if err != nil {
		return nil, err
	}
	rows, err := r.skillRepo.FindByCharacterID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	levels := make(map[string]int, mainSkillFamilyCount)
	for _, row := range rows {
		if row == nil || row.CharacterID != characterID || row.SkillID <= 0 {
			continue
		}
		template := r.catalog.GetSkill(row.SkillID)
		if template == nil {
			continue
		}
		family, exists := families[template.CodeName]
		if !exists {
			continue
		}
		level := exactMainSkillLevel(template)
		if level == 0 || family[level] != template.ID || row.Level != level {
			return nil, fmt.Errorf("invalid persisted main skill %d for character %d", row.SkillID, characterID)
		}
		if level > levels[template.CodeName] {
			levels[template.CodeName] = level
		}
	}
	return []StateProof{
		stateProof(5, mainSkillCountAtLevel(levels, 2), 1),
		stateProof(6, mainSkillCountAtLevel(levels, 5), 3),
		stateProof(7, mainSkillCountAtLevel(levels, 9), 5),
		stateProof(8, mainSkillCountAtLevel(levels, 10), mainSkillFamilyCount),
	}, nil
}

func mainSkillFamilies(templates []*models.SkillTemplate, classID int) (map[string]map[int]int64, error) {
	requiredClass := fmt.Sprintf("|%d|", classID)
	candidates := make(map[string][]*models.SkillTemplate)
	for _, template := range templates {
		level := exactMainSkillLevel(template)
		if level == 0 || template.ReqClass != requiredClass || !isMainSkillTemplate(template) {
			continue
		}
		candidates[template.CodeName] = append(candidates[template.CodeName], template)
	}
	families := make(map[string]map[int]int64)
	for codeName, templates := range candidates {
		if len(templates) != 10 {
			continue
		}
		levels := make(map[int]int64, 10)
		for _, template := range templates {
			level := exactMainSkillLevel(template)
			if _, duplicate := levels[level]; duplicate {
				levels = nil
				break
			}
			levels[level] = template.ID
		}
		levelOneID := levels[1]
		levelOne := findSkillTemplateByID(templates, levelOneID)
		if len(levels) == 10 && levelOne != nil && levelOne.ReqCL == 0 {
			families[codeName] = levels
		}
	}
	if len(families) != mainSkillFamilyCount {
		return nil, fmt.Errorf("class %d main-skill topology has %d families, want %d", classID, len(families), mainSkillFamilyCount)
	}
	return families, nil
}

func findSkillTemplateByID(templates []*models.SkillTemplate, id int64) *models.SkillTemplate {
	for _, template := range templates {
		if template != nil && template.ID == id {
			return template
		}
	}
	return nil
}

func isMainSkillTemplate(template *models.SkillTemplate) bool {
	if template == nil || template.CodeName == "" || template.UseEnv != float64(int(template.UseEnv)) || template.Kind != float64(int(template.Kind)) {
		return false
	}
	return (int(template.UseEnv) == 0 && int(template.Kind) == 1) || (int(template.UseEnv) == 1 && int(template.Kind) == 2)
}

func exactMainSkillLevel(template *models.SkillTemplate) int {
	if template == nil || template.Level != float64(int(template.Level)) || template.Level < 1 || template.Level > 10 {
		return 0
	}
	return int(template.Level)
}

func mainSkillCountAtLevel(levels map[string]int, minimum int) int {
	count := 0
	for _, level := range levels {
		if level >= minimum {
			count++
		}
	}
	return count
}
