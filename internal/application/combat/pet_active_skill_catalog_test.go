// Catalog tests verify every checked-in Pet active skill template has an executable plan.
package combat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type catalogFixture struct {
	Records []map[string]string `json:"records"`
}

func TestPetActiveSkillCatalogCoversAuditedTemplates(t *testing.T) {
	items := readCatalogFixture(t, "TBL_ITEM_TEMPLATE.json")
	skills := readCatalogFixture(t, "TBL_SKILL.json")
	bookNames := make(map[string]bool)
	for _, item := range items.Records {
		if item["type"] == "506" && item["useType"] == "2" {
			bookNames[item["name"]] = true
		}
	}
	covered := 0
	families := make(map[string]bool)
	wantPlans := make(map[string]bool)
	for _, skill := range skills.Records {
		if skill["kind"] != "1" || !bookNames[skill["name"]] {
			continue
		}
		level, err := strconv.Atoi(skill["level"])
		if err != nil {
			t.Fatalf("invalid level for %q: %v", skill["name"], err)
		}
		if petActiveSkillPlan(skill["name"], level) == nil {
			t.Errorf("missing plan for %q level %d", skill["name"], level)
			continue
		}
		covered++
		families[skill["name"]] = true
		wantPlans[skill["name"]+"|"+skill["level"]] = true
	}
	if covered != 144 {
		t.Fatalf("covered templates = %d, want 144", covered)
	}
	if len(families) != 52 {
		t.Fatalf("covered families = %d, want 52", len(families))
	}
	if len(petActiveSkillCatalog) != 52 {
		t.Fatalf("catalog families = %d, want 52", len(petActiveSkillCatalog))
	}
	registered := 0
	for family, levels := range petActiveSkillCatalog {
		for level := range levels {
			registered++
			key := family + "|" + strconv.Itoa(level)
			if !wantPlans[key] {
				t.Errorf("unexpected plan %q level %d", family, level)
			}
		}
	}
	if registered != 144 {
		t.Fatalf("registered plans = %d, want 144", registered)
	}
}

func TestPetActiveSkillCatalogRejectsUnknown(t *testing.T) {
	if plan := petActiveSkillPlan("not a Pet skill", 1); plan != nil {
		t.Fatalf("unknown skill returned plan: %#v", plan)
	}
}

func readCatalogFixture(t *testing.T, name string) catalogFixture {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var fixture catalogFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return fixture
}
