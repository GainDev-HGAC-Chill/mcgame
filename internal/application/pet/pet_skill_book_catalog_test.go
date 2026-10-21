// Catalog tests verify Pet skill-book resolution and exact Super-family mappings.
package pet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type petBookCatalogExport[T any] struct {
	Records []T `json:"records"`
}

type petBookCatalogItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	UseType  string `json:"useType"`
	SkillID  string `json:"skillId"`
	PropType string `json:"propType"`
	ProplNum string `json:"proplNum"`
	ReqClass string `json:"reqClass"`
}

type petBookCatalogSkill struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Level   string `json:"level"`
	UseEnv  string `json:"useEnv"`
	CreKind string `json:"creKind"`
}

func TestPetSkillBookCatalogResolution(t *testing.T) {
	items := readPetBookCatalog[petBookCatalogItem](t, "TBL_ITEM_TEMPLATE.json")
	skillRows := readPetBookCatalog[petBookCatalogSkill](t, "TBL_SKILL.json")
	skills := make([]*models.SkillTemplate, 0, len(skillRows))
	skillByID := make(map[int]*models.SkillTemplate, len(skillRows))
	for _, row := range skillRows {
		skill := &models.SkillTemplate{ID: int64(catalogInt(t, row.ID)), Name: row.Name, Level: float64(catalogInt(t, row.Level)), UseEnv: float64(catalogInt(t, row.UseEnv)), CreKind: row.CreKind}
		skills = append(skills, skill)
		skillByID[skill.GetID()] = skill
	}

	want := map[string]int{
		"Báo Chi Tấn Tiệp": 2741,
		"Cổ Long Chi Hồn":  2820,
		"Ngạnh Xác Hóa":    2696,
		"Vẻ Ngoài Yếu Ớt":  2841,
		"Thụ Bì Ngoại Sáo": 2825,
	}
	bookCount := 0
	resolvedCount := 0
	unresolvedCount := 0
	for _, row := range items {
		if catalogInt(t, row.Type) != 506 || catalogInt(t, row.UseType) != 2 {
			continue
		}
		bookCount++
		book := &models.ItemTemplateTemplate{ID: int64(catalogInt(t, row.ID)), Name: row.Name, SkillID: float64(catalogInt(t, row.SkillID))}
		resolved := resolvePetSkillBookSkillFromCatalog(book, skills, func(id int) *models.SkillTemplate { return skillByID[id] })
		if resolved == nil {
			unresolvedCount++
			continue
		}
		resolvedCount++
		requirements := parsePetSkillRequirements(resolved)
		qLevel := requirements.QLevel
		if qLevel <= 0 {
			qLevel = 1
		}
		pet := &domainpet.Pet{CreatureData: map[string]interface{}{"classId": 1, "classIds": 10, "qLevel": qLevel}}
		if service := NewService(nil, nil); !service.petCanLearnSkillTemplate(pet, resolved) {
			t.Fatalf("book %q resolved skill %d but rejected matching classIds/qLevel", row.Name, resolved.GetID())
		}
		if expectedID, ok := want[row.Name]; ok && resolved.GetID() != expectedID {
			t.Fatalf("book %q resolved skill %d, want %d", row.Name, resolved.GetID(), expectedID)
		}
	}
	if bookCount != 199 || resolvedCount != 154 || unresolvedCount != 45 {
		t.Fatalf("catalog resolution = books %d, resolved %d, unresolved %d; want 199/154/45", bookCount, resolvedCount, unresolvedCount)
	}
	for name := range want {
		found := false
		for _, row := range items {
			if row.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("reported book %q missing from catalog", name)
		}
	}
}

func TestPetSkillEligibilityUsesQualityNotPetFamily(t *testing.T) {
	service := NewService(nil, nil)
	pet := &domainpet.Pet{CreatureData: map[string]interface{}{"classId": 1, "classIds": 10, "qLevel": 3}}
	skill := &models.SkillTemplate{CreKind: "3|3"}
	if !service.petCanLearnSkillTemplate(pet, skill) {
		t.Fatal("different Pet family was rejected despite matching quality")
	}
	pet.CreatureData["qLevel"] = 2
	if service.petCanLearnSkillTemplate(pet, skill) {
		t.Fatal("Pet below required quality was accepted")
	}
}

func TestPetSuperSkillCatalogMatchesExactMappings(t *testing.T) {
	items := readPetBookCatalog[petBookCatalogItem](t, "TBL_ITEM_TEMPLATE.json")
	skills := readPetBookCatalog[petBookCatalogSkill](t, "TBL_SKILL.json")
	itemByID := make(map[int]petBookCatalogItem, len(items))
	for _, item := range items {
		itemByID[catalogInt(t, item.ID)] = item
	}
	skillByID := make(map[int]petBookCatalogSkill, len(skills))
	for _, skill := range skills {
		skillByID[catalogInt(t, skill.ID)] = skill
	}
	if len(petSuperSkillBooks) != 11 {
		t.Fatalf("Super book mappings = %d, want 11", len(petSuperSkillBooks))
	}
	for bookID, spec := range petSuperSkillBooks {
		item, ok := itemByID[bookID]
		if !ok || item.Name != spec.Family+" Siêu Cấp" || catalogInt(t, item.Type) != 506 || catalogInt(t, item.UseType) != 2 || catalogInt(t, item.PropType) != spec.PropType || catalogInt(t, item.ProplNum) != 3 || item.ReqClass != spec.ReqClass {
			t.Fatalf("invalid Super book %d: %#v", bookID, item)
		}
		prerequisiteNames := [3]string{spec.Family, spec.Family + " Trung Cấp", spec.Family + " Cao Cấp"}
		for index, skillID := range spec.Prerequisites {
			skill := skillByID[skillID]
			if skill.Name != prerequisiteNames[index] || catalogInt(t, skill.Level) != 3 || !isPetSkillUseEnvironment(catalogInt(t, skill.UseEnv)) {
				t.Fatalf("invalid prerequisite %d for book %d: %#v", skillID, bookID, skill)
			}
		}
		for index, skillID := range spec.SuperChain {
			skill := skillByID[skillID]
			if skill.Name != spec.Family+" Siêu Cấp" || catalogInt(t, skill.Level) != index+1 || !isPetSkillUseEnvironment(catalogInt(t, skill.UseEnv)) {
				t.Fatalf("invalid Super chain skill %d for book %d: %#v", skillID, bookID, skill)
			}
		}
	}
}

func readPetBookCatalog[T any](t *testing.T, name string) []T {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var export petBookCatalogExport[T]
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatal(err)
	}
	return export.Records
}

func catalogInt(t *testing.T, raw string) int {
	t.Helper()
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("invalid catalog integer %q: %v", raw, err)
	}
	return value
}
