package main

import (
	"testing"

	appquest "mcgame-server/internal/application/quest"
	"mcgame-server/internal/gamedata/models"
)

func TestCheckedCatalogRegistryAudit(t *testing.T) {
	quests, err := loadCheckedQuests("../../docs/database/game_data/TBL_QUEST.json")
	if err != nil {
		t.Fatal(err)
	}
	npcs, err := loadCheckedNPCs("../../docs/database/game_data/TBL_NPC.json")
	if err != nil {
		t.Fatal(err)
	}
	requirements := loadCheckedRequirementPresence(t, "../../docs/database/game_data/TBL_QUEST_REQUIRE.json")
	report := auditRegistry(quests, npcs, requirements)
	if report.Total != 227 || report.CountsByType["1"] != 67 || report.CountsByType["2"] != 67 || report.CountsByType["4"] != 93 {
		t.Fatalf("registry counts=%+v", report)
	}
	if len(report.MissingCatalog) != 0 || len(report.FingerprintMismatches) != 0 || len(report.MissingOrInvalidFinishNPC) != 0 || len(report.RequirementOverlaps) != 0 || len(report.CurrencyOverlaps) != 0 || len(report.MissingDialogue) != 0 {
		t.Fatalf("registry diagnostics=%+v", report)
	}
}

func loadCheckedRequirementPresence(t *testing.T, path string) map[int][]*models.QuestRequireTemplate {
	t.Helper()
	var catalog struct {
		Records []struct {
			Qid string `json:"qid"`
		} `json:"records"`
	}
	if err := readJSON(path, &catalog); err != nil {
		t.Fatal(err)
	}
	result := make(map[int][]*models.QuestRequireTemplate)
	for _, row := range catalog.Records {
		if row.Qid == "" {
			continue
		}
		qid, err := parseInt(row.Qid, "requirement qid")
		if err != nil {
			t.Fatal(err)
		}
		result[qid] = append(result[qid], &models.QuestRequireTemplate{})
	}
	return result
}

func TestRegistryAuditReportsDriftAndOverlaps(t *testing.T) {
	quests := map[int]*models.QuestTemplate{}
	for id, spec := range appquest.ReviewedConversationQuestFingerprints() {
		quests[id] = &models.QuestTemplate{ID: int64(id), Name: spec.Name, Type: float64(spec.QuestType), StartNPC: float64(spec.StartNPC), FinishNPC: float64(spec.FinishNPC), StartText: "start", CompleteText: "complete"}
	}
	quests[46].Name = "drift"
	quests[196].Rt = 1
	quests[71].CompleteText = ""
	npcs := map[int]bool{}
	for _, spec := range appquest.ReviewedConversationQuestFingerprints() {
		npcs[spec.FinishNPC] = true
	}
	delete(npcs, appquest.ReviewedConversationQuestFingerprints()[46].FinishNPC)
	requirements := map[int][]*models.QuestRequireTemplate{46: {{ItemID: 1, Num: 1, Kind: 1, Type: 29}}}
	report := auditRegistry(quests, npcs, requirements)
	if !contains(report.FingerprintMismatches, 46) || !contains(report.MissingOrInvalidFinishNPC, 46) || !contains(report.RequirementOverlaps, 46) || !contains(report.CurrencyOverlaps, 196) || !contains(report.MissingDialogue, 71) {
		t.Fatalf("registry diagnostics=%+v", report)
	}
}

func contains(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
