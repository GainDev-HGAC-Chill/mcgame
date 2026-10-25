// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	appitem "mcgame-server/internal/application/item"
)

type makerSetStateProviderFixture struct {
	result appitem.MakerSetResult
}

func (p makerSetStateProviderFixture) CalculateMakerSet(ctx context.Context, characterID int64) appitem.MakerSetResult {
	return p.result
}

func TestMakerSetStateReaderUsesProductionQualityTier(t *testing.T) {
	tests := []struct {
		name   string
		result appitem.MakerSetResult
		done   []int64
	}{
		{name: "inactive", result: appitem.MakerSetResult{}},
		{name: "mixed orange", result: appitem.MakerSetResult{Active: true, QualityType: 16}, done: []int64{34}},
		{name: "excellent orange", result: appitem.MakerSetResult{Active: true, QualityType: 20}, done: []int64{34, 35}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proofs, err := NewMakerSetStateReader(makerSetStateProviderFixture{result: tt.result}).ReadStateProofs(context.Background(), 99)
			if err != nil {
				t.Fatalf("ReadStateProofs() error = %v", err)
			}
			done := make(map[int64]bool, len(proofs))
			for _, proof := range proofs {
				done[proof.RequirementID] = proof.Done
			}
			want := make(map[int64]bool, len(tt.done))
			for _, requirementID := range tt.done {
				want[requirementID] = true
			}
			for _, requirementID := range []int64{34, 35} {
				if done[requirementID] != want[requirementID] {
					t.Fatalf("requirement %d done = %t, want %t", requirementID, done[requirementID], want[requirementID])
				}
			}
		})
	}
}
