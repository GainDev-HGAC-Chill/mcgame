package main

import (
	"reflect"
	"testing"

	appquest "mcgame-server/internal/application/quest"
	"mcgame-server/internal/gamedata/models"
)

func TestCheckedCatalogRegistryAudit(t *testing.T) {
	snapshot, err := loadSnapshot("../../.local/research/quest-catalog-audit-20261012.json")
	if err != nil {
		t.Fatal(err)
	}
	quests, err := loadCheckedQuests("../../docs/database/game_data/TBL_QUEST.json")
	if err != nil {
		t.Fatal(err)
	}
	npcs, err := loadCheckedNPCs("../../docs/database/game_data/TBL_NPC.json")
	if err != nil {
		t.Fatal(err)
	}
	report, err := buildReport(snapshot, quests, npcs)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 5530 {
		t.Fatalf("total=%d want=5530", report.Total)
	}
	if report.Registry.Total != 227 || report.Registry.CountsByType["1"] != 67 || report.Registry.CountsByType["2"] != 67 || report.Registry.CountsByType["4"] != 93 {
		t.Fatalf("registry counts=%+v", report.Registry)
	}
	if len(report.Registry.MissingCatalog) != 0 || len(report.Registry.FingerprintMismatches) != 0 || len(report.Registry.MissingOrInvalidFinishNPC) != 0 || len(report.Registry.RequirementOverlaps) != 0 || len(report.Registry.CurrencyOverlaps) != 0 || len(report.Registry.MissingDialogue) != 0 {
		t.Fatalf("registry diagnostics=%+v", report.Registry)
	}
	wantCounts := map[string]int{"catalog": 3682, "character_level": 14, "friendship": 39, "missing_target": 4, "pet_count": 23, "pet_level": 14, "reviewed_conversation": 227, "unconfigured": 1312, "unsupported_currency_condition": 215}
	if !reflect.DeepEqual(report.Counts, wantCounts) {
		t.Fatalf("policy counts=%v want=%v", report.Counts, wantCounts)
	}
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
