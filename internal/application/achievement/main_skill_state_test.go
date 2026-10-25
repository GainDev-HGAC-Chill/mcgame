package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	domaincharacter "mcgame-server/internal/domain/character"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata/models"
)

type mainSkillCatalogFixture struct {
	byID map[int]*models.SkillTemplate
	all  []*models.SkillTemplate
}

func (c mainSkillCatalogFixture) GetSkill(id int) *models.SkillTemplate {
	return c.byID[id]
}

func (c mainSkillCatalogFixture) GetAllSkills() []*models.SkillTemplate {
	return c.all
}

func TestMainSkillStateReaderUsesCurrentClassCanonicalFamilies(t *testing.T) {
	catalog := mainSkillCatalogFixture{byID: map[int]*models.SkillTemplate{}}
	rows := make([]*domainskill.CharacterSkill, 0, 5)
	for family := 0; family < mainSkillFamilyCount; family++ {
		codeName := fmt.Sprintf("SKILL111%03d", (family*10)+1)
		for level := 1; level <= 10; level++ {
			id := 10000 + (family * 10) + level
			template := &models.SkillTemplate{ID: int64(id), CodeName: codeName, ReqClass: "|1|", ReqCL: 0, UseEnv: 0, Kind: 1, Level: float64(level)}
			catalog.byID[id] = template
			catalog.all = append(catalog.all, template)
		}
		if family < 5 {
			level := 9
			if family == 0 {
				level = 10
			}
			rows = append(rows, &domainskill.CharacterSkill{CharacterID: 99, SkillID: 10000 + (family * 10) + level, Level: level})
		}
	}
	reader := NewMainSkillStateReader(
		characterStateRepositoryFixture{character: &domaincharacter.Character{ID: 99, ClassID: 1}},
		lifeSkillStateRepositoryFixture{rows: rows},
		catalog,
	)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	for _, proof := range proofs {
		if proof.RequirementID == 8 {
			if proof.Done || proof.Progress != 1 {
				t.Fatalf("requirement 8 proof = %#v", proof)
			}
			continue
		}
		if !proof.Done {
			t.Fatalf("requirement %d proof = %#v, want done", proof.RequirementID, proof)
		}
	}
}

func TestMainSkillStateReaderRejectsIncompleteCatalogTopology(t *testing.T) {
	reader := NewMainSkillStateReader(
		characterStateRepositoryFixture{character: &domaincharacter.Character{ID: 99, ClassID: 1}},
		lifeSkillStateRepositoryFixture{},
		mainSkillCatalogFixture{},
	)
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want topology error")
	}
}

func TestMainSkillFamiliesMatchCheckedInCatalog(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", "TBL_SKILL.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	fixture := achievementCatalogFixture{}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	templates := make([]*models.SkillTemplate, 0, len(fixture.Records))
	for _, record := range fixture.Records {
		input := struct {
			ID       string `json:"id"`
			CodeName string `json:"codeName"`
			ReqClass string `json:"reqClass"`
			ReqCL    string `json:"reqCL"`
			UseEnv   string `json:"useEnv"`
			Kind     string `json:"kind"`
			Level    string `json:"level"`
		}{}
		if err := json.Unmarshal(record, &input); err != nil {
			t.Fatalf("decode skill record: %v", err)
		}
		id, err := strconv.ParseInt(input.ID, 10, 64)
		if err != nil {
			t.Fatalf("parse skill id %q: %v", input.ID, err)
		}
		parseFloat := func(field, value string) float64 {
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				t.Fatalf("parse skill %d %s %q: %v", id, field, value, err)
			}
			return parsed
		}
		templates = append(templates, &models.SkillTemplate{
			ID: id, CodeName: input.CodeName, ReqClass: input.ReqClass,
			ReqCL: parseFloat("reqCL", input.ReqCL), UseEnv: parseFloat("useEnv", input.UseEnv),
			Kind: parseFloat("kind", input.Kind), Level: parseFloat("level", input.Level),
		})
	}
	for classID := 1; classID <= 6; classID++ {
		families, err := mainSkillFamilies(templates, classID)
		if err != nil {
			t.Fatalf("class %d topology: %v", classID, err)
		}
		if len(families) != mainSkillFamilyCount {
			t.Fatalf("class %d families = %d, want %d", classID, len(families), mainSkillFamilyCount)
		}
	}
}
