// Ordinary character-skill catalog tests guard the checked Flash data contract.
package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type normalCatalogFixture struct {
	Records []map[string]string `json:"records"`
}

func TestOrdinarySkillBookCatalogResolvesExactlyAuditedRoster(t *testing.T) {
	items := readNormalCatalogFixture(t, "TBL_ITEM_TEMPLATE.json")
	skills := readNormalCatalogFixture(t, "TBL_SKILL.json")
	resolved := 0
	for _, item := range items.Records {
		if item["type"] != "505" {
			continue
		}
		name, ok := ordinarySkillBookName(item["name"])
		if !ok {
			continue
		}
		candidates := 0
		for _, template := range skills.Records {
			ordinary := (template["useEnv"] == "0" && template["kind"] == "1") ||
				(template["useEnv"] == "1" && template["kind"] == "2")
			if !ordinary || template["level"] != "1" || template["reqClass"] != item["reqClass"] {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(template["name"]), name) {
				candidates++
			}
		}
		if candidates == 1 {
			resolved++
		}
	}
	if resolved != 65 {
		t.Fatalf("resolved ordinary books = %d, want 65", resolved)
	}
}

func readNormalCatalogFixture(t *testing.T, name string) normalCatalogFixture {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var fixture normalCatalogFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return fixture
}
