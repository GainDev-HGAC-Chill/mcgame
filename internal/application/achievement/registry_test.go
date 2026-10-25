// Open-sourced by BaoLT

package achievement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type achievementCatalogFixture struct {
	Records []json.RawMessage `json:"records"`
}

type rawAchievementFixture struct {
	ID     string `json:"id"`
	Enable string `json:"enable"`
	Type   string `json:"type"`
	Kind   string `json:"kind"`
}

type rawRequirementFixture struct {
	ID   string `json:"id"`
	Aid  string `json:"aid"`
	Type string `json:"type"`
	Num  string `json:"num"`
}

func TestRequirementRegistryMatchesAuditedCatalog(t *testing.T) {
	manager := loadAchievementCatalogFixture(t)
	registry, stats, fingerprint, err := BuildRequirementRegistry(manager)
	if err != nil {
		t.Fatalf("BuildRequirementRegistry() error = %v", err)
	}
	if err := ValidateRequirementCatalog(manager); err != nil {
		t.Fatalf("ValidateRequirementCatalog() error = %v", err)
	}
	if registry.Len() != 697 {
		t.Fatalf("registry length = %d, want 697", registry.Len())
	}
	if stats.StateRequirements != 431 || stats.EventRequirements != 159 || stats.UnsupportedRequirements != 107 {
		t.Fatalf("authority stats = %d/%d/%d, want 431/159/107", stats.StateRequirements, stats.EventRequirements, stats.UnsupportedRequirements)
	}
	if stats.EnabledAchievements != 566 || stats.EnabledRequirements != 697 || stats.AllRequirementTypes != 60 {
		t.Fatalf("catalog stats = %#v", stats)
	}
	if stats.MultiRequirementParents != 30 || stats.MultiRequirementRows != 163 || stats.EmptyRequirementParents != 2 {
		t.Fatalf("structure stats = %#v", stats)
	}
	if stats.AuthorityTargetRequirements != 550 {
		t.Fatalf("authority targets = %d, want 550", stats.AuthorityTargetRequirements)
	}
	if fingerprint != auditedAchievementCatalogFingerprint {
		t.Fatalf("fingerprint = %s, want %s", fingerprint, auditedAchievementCatalogFingerprint)
	}
}

func TestRequirementRegistryClassifiesRepresentativeRequirements(t *testing.T) {
	manager := loadAchievementCatalogFixture(t)
	registry, _, _, err := BuildRequirementRegistry(manager)
	if err != nil {
		t.Fatalf("BuildRequirementRegistry() error = %v", err)
	}

	level, ok := registry.Get(1)
	if !ok || level.Authority != RequirementAuthorityState || level.TargetSource != RequirementTargetAuthority || level.AchievementID != 1001 || level.Type != 4 {
		t.Fatalf("level definition = %#v, found=%t", level, ok)
	}
	login, ok := registry.Get(53)
	if !ok || login.Authority != RequirementAuthorityEvent {
		t.Fatalf("login definition = %#v, found=%t", login, ok)
	}
	tower, ok := registry.Get(216)
	if !ok || tower.Authority != RequirementAuthorityUnsupported {
		t.Fatalf("tower definition = %#v, found=%t", tower, ok)
	}
}

func TestRequirementRegistryRejectsCatalogDrift(t *testing.T) {
	manager := loadAchievementCatalogFixture(t)
	requirements := manager.GetAllAchievementRequires()
	for _, requirement := range requirements {
		if requirement != nil && requirement.ID == 1 {
			requirement.Type = 61
			break
		}
	}
	err := ValidateRequirementCatalog(manager)
	if err == nil || !strings.Contains(err.Error(), "fingerprint drift") {
		t.Fatalf("ValidateRequirementCatalog() error = %v, want fingerprint drift", err)
	}
}

func loadAchievementCatalogFixture(t *testing.T) *gamedata.Manager {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadAchievementTableFixture(t, manager, models.TableAchievement, "TBL_ACHIEVEMENT.json")
	loadAchievementTableFixture(t, manager, models.TableAchievementRequire, "TBL_ACHIEVEMENT_REQUIRE.json")
	return manager
}

func loadAchievementTableFixture(t *testing.T, manager *gamedata.Manager, tableName, fileName string) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", fileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	fixture := achievementCatalogFixture{}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	rows := make([]json.RawMessage, 0, len(fixture.Records))
	for _, record := range fixture.Records {
		var model interface{}
		switch tableName {
		case models.TableAchievement:
			input := rawAchievementFixture{}
			if err := json.Unmarshal(record, &input); err != nil {
				t.Fatalf("decode achievement record: %v", err)
			}
			model = models.AchievementTemplate{
				ID:     parseFixtureInt64(t, input.ID),
				Enable: parseFixtureFloat(t, input.Enable),
				Type:   parseFixtureFloat(t, input.Type),
				Kind:   parseFixtureFloat(t, input.Kind),
			}
		case models.TableAchievementRequire:
			input := rawRequirementFixture{}
			if err := json.Unmarshal(record, &input); err != nil {
				t.Fatalf("decode requirement record: %v", err)
			}
			model = models.AchievementRequireTemplate{
				ID:   parseFixtureInt64(t, input.ID),
				Aid:  parseFixtureFloat(t, input.Aid),
				Type: parseFixtureFloat(t, input.Type),
				Num:  parseFixtureFloat(t, input.Num),
			}
		default:
			t.Fatalf("unsupported fixture table %s", tableName)
		}
		encoded, err := json.Marshal(model)
		if err != nil {
			t.Fatalf("encode %s fixture: %v", tableName, err)
		}
		rows = append(rows, encoded)
	}
	if err := manager.GetCache().LoadTable(tableName, rows); err != nil {
		t.Fatalf("load %s: %v", tableName, err)
	}
}

func parseFixtureFloat(t *testing.T, value string) float64 {
	t.Helper()
	if value == "" {
		return 0
	}
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("parse fixture number %q: %v", value, err)
	}
	return result
}

func parseFixtureInt64(t *testing.T, value string) int64 {
	t.Helper()
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("parse fixture integer %q: %v", value, err)
	}
	return result
}
