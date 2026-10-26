// Open-sourced by BaoLT

package achievement

import (
	"encoding/json"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func loadBattleCreatureCatalog(t *testing.T) *gamedata.Manager {
	t.Helper()
	rows := make([]json.RawMessage, 0)
	seen := make(map[int]struct{})
	for _, definition := range battleCreatureRequirements {
		for creatureID, name := range definition.creatures {
			if _, ok := seen[creatureID]; ok {
				continue
			}
			seen[creatureID] = struct{}{}
			raw, err := json.Marshal(models.CreatureTemplate{ID: int64(creatureID), Name: name})
			if err != nil {
				t.Fatalf("marshal creature %d: %v", creatureID, err)
			}
			rows = append(rows, raw)
		}
	}
	manager := gamedata.NewManager(nil, zap.NewNop())
	if err := manager.GetCache().LoadTable(models.TableCreature, rows); err != nil {
		t.Fatalf("load creature catalog: %v", err)
	}
	return manager
}

func TestDefeatedCreatureRequirementsUseExactCatalogIDs(t *testing.T) {
	service := NewService(&achievementTestRepo{}, zap.NewNop())
	service.SetGameDataManager(loadBattleCreatureCatalog(t))

	updates, err := service.defeatedCreatureRequirements([]int{144})
	if err != nil {
		t.Fatalf("defeatedCreatureRequirements() error = %v", err)
	}
	if len(updates) != 2 || updates[0].RequirementID != 190 || updates[1].RequirementID != 194 {
		t.Fatalf("Tà Thần updates = %#v", updates)
	}

	updates, err = service.defeatedCreatureRequirements([]int{1303})
	if err != nil {
		t.Fatalf("partial Mehdi requirements error = %v", err)
	}
	if len(updates) != 0 {
		t.Fatalf("partial Mehdi updates = %#v", updates)
	}
	updates, err = service.defeatedCreatureRequirements([]int{1303, 1304})
	if err != nil {
		t.Fatalf("complete Mehdi requirements error = %v", err)
	}
	if len(updates) != 1 || updates[0].RequirementID != 530 {
		t.Fatalf("complete Mehdi updates = %#v", updates)
	}
}

func TestDefeatedCreatureRequirementsFailClosedOnCatalogDrift(t *testing.T) {
	manager := loadBattleCreatureCatalog(t)
	creature := manager.GetCreature(144)
	creature.Name = "drifted"
	service := NewService(&achievementTestRepo{}, zap.NewNop())
	service.SetGameDataManager(manager)
	if _, err := service.defeatedCreatureRequirements([]int{144}); err == nil {
		t.Fatal("defeatedCreatureRequirements() error = nil, want catalog drift rejection")
	}
}
