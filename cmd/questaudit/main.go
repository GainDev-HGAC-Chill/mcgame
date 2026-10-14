// Read-only catalog policy audit. Inputs are checked catalog data and the compose query snapshot, never player data.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	appquest "mcgame-server/internal/application/quest"
	"mcgame-server/internal/gamedata/models"
)

type catalogRow struct {
	ID             int64                          `json:"id"`
	Requirements   []*models.QuestRequireTemplate `json:"requirements"`
	MissingTargets int                            `json:"missing_targets"`
}

type checkedQuestRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	StartNPC     string `json:"startNpc"`
	FinishNPC    string `json:"finishNpc"`
	StartText    string `json:"startText"`
	CompleteText string `json:"completeText"`
	Rt           string `json:"rt"`
	Rn           string `json:"rn"`
}

type checkedNPCRow struct {
	ID string `json:"id"`
}

type policyRow struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Policy          string `json:"policy"`
	Threshold       int    `json:"threshold,omitempty"`
	RequirementRows int    `json:"requirement_rows"`
}

type registryReport struct {
	Total                     int            `json:"total"`
	CountsByType              map[string]int `json:"counts_by_type"`
	MissingCatalog            []int          `json:"missing_catalog,omitempty"`
	FingerprintMismatches     []int          `json:"fingerprint_mismatches,omitempty"`
	MissingOrInvalidFinishNPC []int          `json:"missing_or_invalid_finish_npc,omitempty"`
	RequirementOverlaps       []int          `json:"requirement_overlaps,omitempty"`
	CurrencyOverlaps          []int          `json:"currency_overlaps,omitempty"`
	MissingDialogue           []int          `json:"missing_dialogue,omitempty"`
}

type auditReport struct {
	Total    int            `json:"total"`
	Counts   map[string]int `json:"counts"`
	Registry registryReport `json:"reviewed_conversation_registry"`
	Rows     []policyRow    `json:"rows"`
}

func main() {
	input := flag.String("input", ".local/research/quest-catalog-audit-20261012.json", "compose CLI snapshot")
	catalog := flag.String("catalog", "docs/database/game_data/TBL_QUEST.json", "checked quest catalog")
	npcs := flag.String("npc-catalog", "docs/database/game_data/TBL_NPC.json", "checked NPC catalog")
	output := flag.String("output", ".local/research/quest-completion-policy-20261012.json", "policy audit artifact")
	flag.Parse()
	if err := run(*input, *catalog, *npcs, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input, catalogPath, npcPath, output string) error {
	snapshotRows, err := loadSnapshot(input)
	if err != nil {
		return err
	}
	quests, err := loadCheckedQuests(catalogPath)
	if err != nil {
		return err
	}
	npcs, err := loadCheckedNPCs(npcPath)
	if err != nil {
		return err
	}
	report, err := buildReport(snapshotRows, quests, npcs)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, encoded, 0600); err != nil {
		return err
	}
	summary, _ := json.Marshal(report.Counts)
	fmt.Printf("Audited %d unique quests: %s\n", report.Total, summary)
	fmt.Printf("Reviewed conversation registry: total=%d type1=%d type2=%d type4=%d drift=%d invalid_finish_npc=%d requirement_overlap=%d currency_overlap=%d missing_dialogue=%d\n",
		report.Registry.Total,
		report.Registry.CountsByType["1"],
		report.Registry.CountsByType["2"],
		report.Registry.CountsByType["4"],
		len(report.Registry.MissingCatalog)+len(report.Registry.FingerprintMismatches),
		len(report.Registry.MissingOrInvalidFinishNPC),
		len(report.Registry.RequirementOverlaps),
		len(report.Registry.CurrencyOverlaps),
		len(report.Registry.MissingDialogue),
	)
	return nil
}

func buildReport(snapshotRows []catalogRow, quests map[int]*models.QuestTemplate, npcs map[int]bool) (*auditReport, error) {
	counts := make(map[string]int)
	seen := make(map[int64]bool)
	requirements := make(map[int][]*models.QuestRequireTemplate)
	rows := make([]policyRow, 0, len(snapshotRows))
	for _, row := range snapshotRows {
		if row.ID <= 0 || seen[row.ID] {
			return nil, fmt.Errorf("invalid/duplicate quest id %d", row.ID)
		}
		seen[row.ID] = true
		tpl := quests[int(row.ID)]
		if tpl == nil {
			return nil, fmt.Errorf("quest %d missing from checked catalog", row.ID)
		}
		requirements[int(row.ID)] = row.Requirements
		policy, threshold := appquest.ClassifyQuestCompletion(tpl, row.Requirements)
		if policy == appquest.PolicyCatalog && row.MissingTargets > 0 {
			policy = "missing_target"
		}
		counts[policy]++
		rows = append(rows, policyRow{ID: row.ID, Name: tpl.Name, Policy: policy, Threshold: threshold, RequirementRows: len(row.Requirements)})
	}
	registry := auditRegistry(quests, npcs, requirements)
	return &auditReport{Total: len(rows), Counts: counts, Registry: registry, Rows: rows}, nil
}

func auditRegistry(quests map[int]*models.QuestTemplate, npcs map[int]bool, requirements map[int][]*models.QuestRequireTemplate) registryReport {
	fingerprints := appquest.ReviewedConversationQuestFingerprints()
	report := registryReport{Total: len(fingerprints), CountsByType: make(map[string]int)}
	for id, spec := range fingerprints {
		report.CountsByType[strconv.Itoa(spec.QuestType)]++
		tpl := quests[id]
		if tpl == nil {
			report.MissingCatalog = append(report.MissingCatalog, id)
			continue
		}
		if tpl.Type != float64(spec.QuestType) || tpl.StartNPC != float64(spec.StartNPC) || tpl.FinishNPC != float64(spec.FinishNPC) || tpl.Name != spec.Name {
			report.FingerprintMismatches = append(report.FingerprintMismatches, id)
		}
		finishNPC := int(tpl.FinishNPC)
		if tpl.FinishNPC <= 0 || tpl.FinishNPC != float64(finishNPC) || !npcs[finishNPC] {
			report.MissingOrInvalidFinishNPC = append(report.MissingOrInvalidFinishNPC, id)
		}
		if len(requirements[id]) > 0 {
			report.RequirementOverlaps = append(report.RequirementOverlaps, id)
		}
		if tpl.Rt != 0 || tpl.Rn != 0 {
			report.CurrencyOverlaps = append(report.CurrencyOverlaps, id)
		}
		if strings.TrimSpace(tpl.StartText) == "" || strings.TrimSpace(tpl.CompleteText) == "" {
			report.MissingDialogue = append(report.MissingDialogue, id)
		}
	}
	sortRegistryReport(&report)
	return report
}

func sortRegistryReport(report *registryReport) {
	sort.Ints(report.MissingCatalog)
	sort.Ints(report.FingerprintMismatches)
	sort.Ints(report.MissingOrInvalidFinishNPC)
	sort.Ints(report.RequirementOverlaps)
	sort.Ints(report.CurrencyOverlaps)
	sort.Ints(report.MissingDialogue)
}

func loadSnapshot(path string) ([]catalogRow, error) {
	var snapshot struct {
		Rows []catalogRow `json:"rows"`
	}
	if err := readJSON(path, &snapshot); err != nil {
		return nil, err
	}
	return snapshot.Rows, nil
}

func loadCheckedQuests(path string) (map[int]*models.QuestTemplate, error) {
	var catalog struct {
		Records []checkedQuestRow `json:"records"`
	}
	if err := readJSON(path, &catalog); err != nil {
		return nil, err
	}
	result := make(map[int]*models.QuestTemplate, len(catalog.Records))
	for _, row := range catalog.Records {
		id, err := parseInt(row.ID, "quest id")
		if err != nil {
			return nil, err
		}
		if result[id] != nil {
			return nil, fmt.Errorf("duplicate checked quest id %d", id)
		}
		kind, err := parseNumber(row.Type, id, "type")
		if err != nil {
			return nil, err
		}
		startNPC, err := parseNumber(row.StartNPC, id, "start NPC")
		if err != nil {
			return nil, err
		}
		finishNPC, err := parseNumber(row.FinishNPC, id, "finish NPC")
		if err != nil {
			return nil, err
		}
		rt, err := parseNumber(row.Rt, id, "rt")
		if err != nil {
			return nil, err
		}
		rn, err := parseNumber(row.Rn, id, "rn")
		if err != nil {
			return nil, err
		}
		result[id] = &models.QuestTemplate{ID: int64(id), Name: row.Name, Type: kind, StartNPC: startNPC, FinishNPC: finishNPC, StartText: row.StartText, CompleteText: row.CompleteText, Rt: rt, Rn: rn}
	}
	return result, nil
}

func loadCheckedNPCs(path string) (map[int]bool, error) {
	var catalog struct {
		Records []checkedNPCRow `json:"records"`
	}
	if err := readJSON(path, &catalog); err != nil {
		return nil, err
	}
	result := make(map[int]bool, len(catalog.Records))
	for _, row := range catalog.Records {
		id, err := parseInt(row.ID, "NPC id")
		if err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, nil
}

func readJSON(path string, dst interface{}) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(content) >= 3 && string(content[:3]) == "\xef\xbb\xbf" {
		content = content[3:]
	}
	return json.Unmarshal(content, dst)
}

func parseInt(value, field string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q: %w", field, value, err)
	}
	return n, nil
}

func parseNumber(value string, id int, field string) (float64, error) {
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("quest %d %s %q: %w", id, field, value, err)
	}
	return n, nil
}
