// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
)

type marriageHistoryStateRepositoryFixture struct {
	normalWeddings     int
	marriageCount      int
	currentMarriedDays int
	err                error
}

func (r marriageHistoryStateRepositoryFixture) GetMarriageAchievementState(context.Context, int64) (int, int, int, error) {
	return r.normalWeddings, r.marriageCount, r.currentMarriedDays, r.err
}

func TestMarriageHistoryStateReaderUsesDurableAggregate(t *testing.T) {
	reader := NewMarriageHistoryStateReader(marriageHistoryStateRepositoryFixture{
		normalWeddings: 1, marriageCount: 4, currentMarriedDays: 365,
	})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	done := make(map[int64]bool, len(proofs))
	for _, proof := range proofs {
		done[proof.RequirementID] = proof.Done
	}
	for _, requirementID := range []int64{131, 346, 356, 357, 358} {
		if !done[requirementID] {
			t.Fatalf("requirement %d should be done: %#v", requirementID, proofs)
		}
	}
	for _, requirementID := range []int64{347, 359} {
		if done[requirementID] {
			t.Fatalf("requirement %d should be incomplete: %#v", requirementID, proofs)
		}
	}
}

func TestMarriageHistoryStateReaderRejectsInconsistentAggregate(t *testing.T) {
	reader := NewMarriageHistoryStateReader(marriageHistoryStateRepositoryFixture{normalWeddings: 2, marriageCount: 1})
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want inconsistent aggregate rejection")
	}
}
