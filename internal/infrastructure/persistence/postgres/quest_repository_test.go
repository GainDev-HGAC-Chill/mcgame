package postgres

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/domain/quest"
)

func TestDecodeQuestObjectivesInvalidJSON(t *testing.T) {
	for _, tc := range []struct {
		name      string
		input     string
		typeError bool
	}{
		{"missing", "", false},
		{"whitespace", "  ", false},
		{"truncated", `[{"type":1`, false},
		{"object", `{"private":"objective-secret"}`, true},
		{"string", `"objective-secret"`, true},
		{"number", `42`, true},
		{"boolean", `true`, true},
		{"element type", `["objective-secret"]`, true},
		{"field type", `[{"type":1,"required":"objective-secret"}]`, true},
		{"partial decode", `[{"type":1,"required":5},{"required":"objective-secret"}]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := []quest.Objective{{Type: quest.ObjectiveKillMonster, Required: 5}}
			qp := &quest.QuestProgress{ID: 101, CharacterID: 202, QuestID: 303, Objectives: original}
			err := decodeQuestObjectives(qp, []byte(tc.input))
			if err == nil {
				t.Fatal("invalid objectives must return an error")
			}
			for _, identity := range []string{"progress ID 101", "character ID 202", "quest ID 303"} {
				if !strings.Contains(err.Error(), identity) {
					t.Fatalf("missing identity %q: %v", identity, err)
				}
			}
			if strings.Contains(err.Error(), "objective-secret") {
				t.Fatalf("error contains objective content: %v", err)
			}
			if !reflect.DeepEqual(qp.Objectives, original) {
				t.Fatal("failed decode changed objectives")
			}
			if tc.typeError {
				var target *json.UnmarshalTypeError
				if !errors.As(err, &target) {
					t.Fatalf("type error not wrapped: %v", err)
				}
			} else {
				var target *json.SyntaxError
				if !errors.As(err, &target) {
					t.Fatalf("syntax error not wrapped: %v", err)
				}
			}
		})
	}
}

func TestDecodeQuestObjectivesValidJSON(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  []quest.Objective
	}{
		{"empty array", `[]`, []quest.Objective{}},
		{"null preserves nil", `null`, nil},
		{"null with whitespace", " \n null \t", nil},
		{"objectives", `[{"type":2,"target":17,"current":1,"required":5,"itemTableType":29,"quality":3}]`, []quest.Objective{{Type: quest.ObjectiveCollectItem, Target: 17, Current: 1, Required: 5, ItemTableType: 29, Quality: 3}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			qp := &quest.QuestProgress{Objectives: []quest.Objective{{Required: 99}}}
			if err := decodeQuestObjectives(qp, []byte(tc.input)); err != nil {
				t.Fatalf("valid objectives rejected: %v", err)
			}
			if !reflect.DeepEqual(qp.Objectives, tc.want) {
				t.Fatalf("objectives = %#v, want %#v", qp.Objectives, tc.want)
			}
		})
	}
}

type questObjectiveRows struct {
	pgx.Rows
	inputs []string
	index  int
}

func (r *questObjectiveRows) Next() bool {
	return r.index < len(r.inputs)
}

func (r *questObjectiveRows) Scan(dest ...any) error {
	*dest[0].(*int64) = int64(101 + r.index)
	*dest[1].(*int64) = 202
	*dest[2].(*int) = 303
	*dest[3].(*int) = int(quest.QuestStatusActive)
	*dest[4].(*[]byte) = []byte(r.inputs[r.index])
	r.index++
	return nil
}

func TestScanQuestRowsRejectsInvalidObjectives(t *testing.T) {
	rows := &questObjectiveRows{inputs: []string{`[]`, `{"private":"objective-secret"}`}}
	repo := &QuestRepository{}
	quests, err := repo.scanQuestRows(rows)
	if err == nil || quests != nil {
		t.Fatalf("malformed row returned progress: quests=%v, err=%v", quests, err)
	}
	if !strings.Contains(err.Error(), "progress ID 102") {
		t.Fatalf("error lacks malformed row identity: %v", err)
	}
}

func TestScanQuestRowsPreservesEmptyObjectives(t *testing.T) {
	rows := &questObjectiveRows{inputs: []string{`[]`, `null`}}
	repo := &QuestRepository{}
	quests, err := repo.scanQuestRows(rows)
	if err != nil || len(quests) != 2 {
		t.Fatalf("valid empty rows rejected: quests=%v, err=%v", quests, err)
	}
	if quests[0].Objectives == nil || len(quests[0].Objectives) != 0 || quests[1].Objectives != nil {
		t.Fatal("empty array and null semantics changed")
	}
}
