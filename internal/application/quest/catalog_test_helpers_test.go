package quest

import (
	"encoding/json"
	"testing"

	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
)

func loadTestQuestRequirement(t *testing.T, manager *gamedata.Manager, row models.QuestRequireTemplate) {
	t.Helper()
	if row.Type == 12 && manager.GetCreature(int(row.ItemID)) == nil {
		if err := manager.GetCache().LoadTable(models.TableCreature, []json.RawMessage{mustQuestJSON(t, models.CreatureTemplate{ID: int64(row.ItemID), Name: "Test creature"})}); err != nil {
			t.Fatal(err)
		}
	}
	if row.Type == 29 && manager.GetItem(int(row.ItemID)) == nil {
		if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{mustQuestJSON(t, models.ItemTemplateTemplate{ID: int64(row.ItemID), Name: "Test item", StackMax: 99})}); err != nil {
			t.Fatal(err)
		}
	}
	if row.Type == 19 && manager.GetEquipment(int(row.ItemID)) == nil {
		if err := manager.GetCache().LoadTable(models.TableEquiptTemplate, []json.RawMessage{mustQuestJSON(t, models.EquiptTemplateTemplate{ID: int64(row.ItemID), Name: "Test equipment"})}); err != nil {
			t.Fatal(err)
		}
	}
	if row.ID == 0 {
		row.ID = int64(row.Qid)
	}
	if err := manager.GetCache().LoadTable(models.TableQuestRequire, []json.RawMessage{mustQuestJSON(t, row)}); err != nil {
		t.Fatal(err)
	}
}

func loadTestCompletedKill(t *testing.T, manager *gamedata.Manager, questID int) []domainquest.Objective {
	t.Helper()
	loadTestQuestRequirement(t, manager, models.QuestRequireTemplate{Qid: float64(questID), ItemID: 9001, Kind: 2, Type: 12, Num: 1})
	return []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 1, Current: 1}}
}
