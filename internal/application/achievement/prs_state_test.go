// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
)

type prsStateProviderFixture struct {
	panel map[string]interface{}
}

func (p prsStateProviderFixture) GetPanel(ctx context.Context, characterID int64) (map[string]interface{}, error) {
	return p.panel, nil
}

func TestPRSStateReaderMapsPermanentAndLimitedForms(t *testing.T) {
	provider := prsStateProviderFixture{panel: map[string]interface{}{
		"actArr": map[string]interface{}{
			"1": 1,
			"2": float64(10),
			"3": 17,
		},
		"actLimitObj": map[string]interface{}{
			"12": int64(123),
			"18": int64(456),
		},
	}}
	proofs, err := NewPRSStateReader(provider).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 15 {
		t.Fatalf("proof count = %d, want 15", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{715, 725, 735, 758, 759} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{716, 734, 736} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

var _ PRSStateProvider = prsStateProviderFixture{}
