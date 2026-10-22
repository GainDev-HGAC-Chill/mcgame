package quest

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"mcgame-server/internal/gamedata/models"
)

type checkedQuestCatalog struct {
	Records []checkedQuestRecord `json:"records"`
}

type checkedQuestRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	StartNPC     string `json:"startNpc"`
	FinishNPC    string `json:"finishNpc"`
	StartText    string `json:"startText"`
	CompleteText string `json:"completeText"`
	ReqClass     string `json:"reqClass"`
	Rt           string `json:"rt"`
	Rn           string `json:"rn"`
}

func TestClassRankRegistryMatchesCheckedCatalog(t *testing.T) {
	content, err := os.ReadFile("../../../docs/database/game_data/TBL_QUEST.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog checkedQuestCatalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	byID := make(map[int]checkedQuestRecord, len(catalog.Records))
	for _, row := range catalog.Records {
		id, err := strconv.Atoi(row.ID)
		if err != nil {
			t.Fatalf("quest id %q: %v", row.ID, err)
		}
		byID[id] = row
	}
	if len(reviewedClassRankQuests) != 6 {
		t.Fatalf("registry count=%d want=6", len(reviewedClassRankQuests))
	}
	for id, spec := range reviewedClassRankQuests {
		row, ok := byID[id]
		if !ok {
			t.Fatalf("quest %d missing from checked catalog", id)
		}
		tpl := checkedRecordTemplate(t, row)
		if !classRankQuestFingerprintMatches(tpl, spec) {
			t.Fatalf("quest %d registry drift: spec=%+v catalog=%+v", id, spec, tpl)
		}
		if policy, rank := ClassifyQuestCompletion(tpl, nil); policy != PolicyClassRank || rank != 1 {
			t.Fatalf("quest %d policy=%s rank=%d", id, policy, rank)
		}
	}
}

func TestConversationRegistryMatchesCheckedCatalog(t *testing.T) {
	content, err := os.ReadFile("../../../docs/database/game_data/TBL_QUEST.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog checkedQuestCatalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	byID := make(map[int]checkedQuestRecord, len(catalog.Records))
	for _, row := range catalog.Records {
		id, err := strconv.Atoi(row.ID)
		if err != nil {
			t.Fatalf("quest id %q: %v", row.ID, err)
		}
		byID[id] = row
	}
	counts := make(map[int]int)
	if len(reviewedConversationQuests) != 242 {
		t.Fatalf("registry count=%d want=242", len(reviewedConversationQuests))
	}
	for id, spec := range reviewedConversationQuests {
		row, ok := byID[id]
		if !ok {
			t.Fatalf("quest %d missing from checked catalog", id)
		}
		tpl := checkedRecordTemplate(t, row)
		if tpl.Name != spec.Name || tpl.Type != float64(spec.QuestType) || tpl.StartNPC != float64(spec.StartNPC) || tpl.FinishNPC != float64(spec.FinishNPC) || (spec.ClassID > 0 && tpl.ReqClass != "|"+strconv.Itoa(spec.ClassID)+"|") {
			t.Fatalf("quest %d registry drift: spec=%+v catalog=%+v", id, spec, tpl)
		}
		if policy, target := ClassifyQuestCompletion(tpl, nil); policy != PolicyConversation || target != spec.FinishNPC {
			t.Fatalf("quest %d policy=%s target=%d", id, policy, target)
		}
		counts[spec.QuestType]++
	}
	if counts[1] != 75 || counts[2] != 67 || counts[4] != 94 || counts[10] != 6 || len(counts) != 4 {
		t.Fatalf("registry type counts=%v", counts)
	}
}

func TestConversationPolicyExactFingerprintAndPrecedence(t *testing.T) {
	base := models.QuestTemplate{ID: 46, Name: "Chế Tạo Huy Hiệu", Type: 1, StartNPC: 430, FinishNPC: 430, StartText: "start", CompleteText: "complete"}
	if policy, target := ClassifyQuestCompletion(&base, nil); policy != PolicyConversation || target != 430 {
		t.Fatalf("quest 46 policy=%s target=%d", policy, target)
	}
	for _, tc := range []struct {
		name   string
		change func(*models.QuestTemplate)
	}{
		{"name", func(q *models.QuestTemplate) { q.Name += " drift" }},
		{"type", func(q *models.QuestTemplate) { q.Type = 2 }},
		{"start NPC", func(q *models.QuestTemplate) { q.StartNPC++ }},
		{"finish NPC", func(q *models.QuestTemplate) { q.FinishNPC++ }},
		{"fractional finish NPC", func(q *models.QuestTemplate) { q.FinishNPC += 0.5 }},
		{"start dialogue", func(q *models.QuestTemplate) { q.StartText = " " }},
		{"complete dialogue", func(q *models.QuestTemplate) { q.CompleteText = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tpl := base
			tc.change(&tpl)
			if policy, _ := ClassifyQuestCompletion(&tpl, nil); policy != PolicyUnconfigured {
				t.Fatalf("drift policy=%s", policy)
			}
		})
	}
	requirement := &models.QuestRequireTemplate{ItemID: 1, Num: 1, Kind: 1, Type: 29}
	if policy, _ := ClassifyQuestCompletion(&base, []*models.QuestRequireTemplate{requirement}); policy != PolicyCatalog {
		t.Fatalf("ordinary requirement policy=%s", policy)
	}
	rt := base
	rt.Rt = 1
	if policy, _ := ClassifyQuestCompletion(&rt, []*models.QuestRequireTemplate{requirement}); policy != PolicyCatalog {
		t.Fatalf("ordinary requirement precedence policy=%s", policy)
	}
	if policy, _ := ClassifyQuestCompletion(&rt, nil); policy != PolicyCurrency {
		t.Fatalf("empty rt policy=%s", policy)
	}
	rn := base
	rn.Rn = 1
	if policy, _ := ClassifyQuestCompletion(&rn, []*models.QuestRequireTemplate{requirement}); policy != PolicyCurrency {
		t.Fatalf("rn precedence policy=%s", policy)
	}
	ambiguous := models.QuestTemplate{ID: 6001, Name: "Ambiguous", Type: 3, StartNPC: 1, FinishNPC: 2, StartText: "start", CompleteText: "complete"}
	if policy, _ := ClassifyQuestCompletion(&ambiguous, nil); policy != PolicyUnconfigured {
		t.Fatalf("ambiguous policy=%s", policy)
	}
}

func checkedRecordTemplate(t *testing.T, row checkedQuestRecord) *models.QuestTemplate {
	t.Helper()
	values := make([]float64, 5)
	texts := []string{row.Type, row.StartNPC, row.FinishNPC, row.Rt, row.Rn}
	for i := range texts {
		value, err := strconv.ParseFloat(texts[i], 64)
		if err != nil {
			t.Fatalf("quest %s numeric field %q: %v", row.ID, texts[i], err)
		}
		values[i] = value
	}
	id, err := strconv.ParseInt(row.ID, 10, 64)
	if err != nil {
		t.Fatalf("quest id %q: %v", row.ID, err)
	}
	return &models.QuestTemplate{ID: id, Name: row.Name, Type: values[0], StartNPC: values[1], FinishNPC: values[2], ReqClass: row.ReqClass, Rt: values[3], Rn: values[4], StartText: row.StartText, CompleteText: row.CompleteText}
}
