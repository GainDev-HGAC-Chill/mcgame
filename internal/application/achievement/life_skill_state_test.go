// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	appskill "mcgame-server/internal/application/skill"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata/models"
)

type lifeSkillStateRepositoryFixture struct {
	rows []*domainskill.CharacterSkill
}

func (r lifeSkillStateRepositoryFixture) FindByCharacterID(ctx context.Context, characterID int64) ([]*domainskill.CharacterSkill, error) {
	return r.rows, nil
}

type lifeSkillCatalogFixture map[int]*models.SkillTemplate

func (c lifeSkillCatalogFixture) GetSkill(id int) *models.SkillTemplate {
	return c[id]
}

func TestLifeSkillStateReaderUsesOwnedSkillsAndExactRecipes(t *testing.T) {
	catalog := lifeSkillCatalogFixture{}
	rows := []*domainskill.CharacterSkill{}
	add := func(skillID, skillType, level int) {
		catalog[skillID] = &models.SkillTemplate{ID: int64(skillID), Type: float64(skillType)}
		rows = append(rows, &domainskill.CharacterSkill{CharacterID: 99, SkillID: skillID, Level: level})
	}
	add(100, appskill.PlantSkillType, 6)
	add(101, appskill.HerbSkillType, 1)
	add(102, appskill.CookbookSkillType, 1)
	add(5048, appskill.MedicinebookSkillType, 1)
	add(5059, appskill.MedicinebookSkillType, 5)
	add(999, appskill.CookbookSkillType, 1)
	rows = append(rows, &domainskill.CharacterSkill{CharacterID: 100, SkillID: 200, Level: 10})
	catalog[200] = &models.SkillTemplate{ID: 200, Type: appskill.FishSkillType}

	proofs, err := NewLifeSkillStateReader(lifeSkillStateRepositoryFixture{rows: rows}, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 14 {
		t.Fatalf("proof count = %d, want 14", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if byID[276].Progress != 2 || byID[276].Done {
		t.Fatalf("cookbook proof = %#v", byID[276])
	}
	for _, requirementID := range []int64{281, 285, 295, 296, 297, 301} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{282, 283, 284, 286, 298, 299, 300} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

var _ LifeSkillStateRepository = lifeSkillStateRepositoryFixture{}
var _ LifeSkillCatalog = lifeSkillCatalogFixture{}
