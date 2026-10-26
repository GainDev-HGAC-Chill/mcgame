// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	domainachievement "mcgame-server/internal/domain/achievement"
)

type battleDerivedStateRepositoryFixture struct {
	rows []*domainachievement.RequirementProgress
}

func (r battleDerivedStateRepositoryFixture) ListRequirementProgress(context.Context, int64) ([]*domainachievement.RequirementProgress, error) {
	return r.rows, nil
}

func TestBattleDerivedStateReaderRequiresEveryPersistedSubRequirement(t *testing.T) {
	rows := make([]*domainachievement.RequirementProgress, 0)
	for _, requirementID := range []int64{194, 195, 196, 197, 202, 203, 204} {
		rows = append(rows, &domainachievement.RequirementProgress{RequirementID: requirementID, Done: true})
	}
	proofs, err := NewBattleDerivedStateReader(battleDerivedStateRepositoryFixture{rows: rows}).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 2 || !proofs[0].Done || proofs[1].Done {
		t.Fatalf("proofs = %#v", proofs)
	}
}
