// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
)

type questStateRepositoryFixture struct {
	completed []int
}

func (r questStateRepositoryFixture) GetCompletedQuestIDs(ctx context.Context, characterID int64) ([]int, error) {
	return r.completed, nil
}

func TestQuestStateReaderMapsOnlyExactAuditedQuestIDs(t *testing.T) {
	reader := NewQuestStateReader(questStateRepositoryFixture{completed: []int{689, 677, 2471, 4045, 170, 1299, 3380, 3380, 5414, -1}})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 12 {
		t.Fatalf("proof count = %d, want 12", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{259, 260, 261, 262, 263, 266, 268, 511} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{264, 265, 267, 269} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

var _ QuestStateRepository = questStateRepositoryFixture{}
