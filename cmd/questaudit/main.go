// Read-only catalog policy audit. Input is the compose Supabase CLI query snapshot, not player data.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	appquest "mcgame-server/internal/application/quest"
	"mcgame-server/internal/gamedata/models"
)

type catalogRow struct {
	ID             int64                          `json:"id"`
	Name           string                         `json:"name"`
	Type           json.Number                    `json:"type"`
	Info           string                         `json:"info"`
	FinishNPC      json.Number                    `json:"finish_npc"`
	Rn             json.Number                    `json:"rn"`
	Requirements   []*models.QuestRequireTemplate `json:"requirements"`
	MissingTargets int                            `json:"missing_targets"`
}

type policyRow struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Policy          string `json:"policy"`
	Threshold       int    `json:"threshold,omitempty"`
	RequirementRows int    `json:"requirement_rows"`
}

func main() {
	input := flag.String("input", ".local/research/quest-catalog-audit-20261012.json", "compose CLI snapshot")
	output := flag.String("output", ".local/research/quest-completion-policy-20261012.json", "policy audit artifact")
	flag.Parse()
	if err := run(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input, output string) error {
	content, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	var snapshot struct {
		Rows []catalogRow `json:"rows"`
	}
	if err := json.Unmarshal(content, &snapshot); err != nil {
		if len(content) >= 3 && string(content[:3]) == "\xef\xbb\xbf" {
			err = json.Unmarshal(content[3:], &snapshot)
		}
		if err != nil {
			return err
		}
	}
	counts := make(map[string]int)
	seen := make(map[int64]bool)
	rows := make([]policyRow, 0, len(snapshot.Rows))
	for _, row := range snapshot.Rows {
		if row.ID <= 0 || seen[row.ID] {
			return fmt.Errorf("invalid/duplicate quest id %d", row.ID)
		}
		seen[row.ID] = true
		kind, err := row.Type.Float64()
		if err != nil {
			return fmt.Errorf("quest %d type: %w", row.ID, err)
		}
		finish, err := row.FinishNPC.Float64()
		if err != nil {
			return fmt.Errorf("quest %d finish NPC: %w", row.ID, err)
		}
		rn, err := row.Rn.Float64()
		if err != nil {
			return fmt.Errorf("quest %d rn: %w", row.ID, err)
		}
		tpl := &models.QuestTemplate{ID: row.ID, Name: row.Name, Info: row.Info, Type: kind, FinishNPC: finish, Rn: rn}
		policy, n := appquest.ClassifyQuestCompletion(tpl, row.Requirements)
		if policy == appquest.PolicyCatalog && row.MissingTargets > 0 {
			policy = "missing_target"
		}
		counts[policy]++
		rows = append(rows, policyRow{ID: row.ID, Name: row.Name, Policy: policy, Threshold: n, RequirementRows: len(row.Requirements)})
	}
	report := struct {
		Total  int            `json:"total"`
		Counts map[string]int `json:"counts"`
		Rows   []policyRow    `json:"rows"`
	}{len(rows), counts, rows}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, encoded, 0600); err != nil {
		return err
	}
	summary, _ := json.Marshal(counts)
	fmt.Printf("Audited %d unique quests: %s\n", len(rows), summary)
	return nil
}
