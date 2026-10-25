// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
)

type starInstanceStateProviderFixture struct {
	grid map[string]any
}

func (p starInstanceStateProviderFixture) WarMapLoginGrid(ctx context.Context, characterID int64) (map[string]any, error) {
	return p.grid, nil
}

func TestStarInstanceStateReaderMapsRequirementGrid(t *testing.T) {
	provider := starInstanceStateProviderFixture{grid: map[string]any{
		"1":  map[string]any{"1": float64(80), "12": float64(50)},
		"6":  map[string]any{"1": float64(70), "12": float64(60)},
		"12": map[string]any{"1": float64(100), "12": float64(90)},
	}}
	proofs, err := NewStarInstanceStateReader(provider).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 73 {
		t.Fatalf("proof count = %d, want 73", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{425, 426, 437, 438, 449, 450, 461, 463, 464, 485, 486, 487, 498} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{427, 439, 451, 465, 488} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
	if _, exists := byID[462]; exists {
		t.Fatalf("unsupported requirement 462 must not be emitted")
	}
}

var _ StarInstanceStateProvider = starInstanceStateProviderFixture{}
