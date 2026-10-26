// Open-sourced by BaoLT

// Derived battle state proves grouped boss milestones from persisted exact sub-requirements.
package achievement

import (
	"context"
	"fmt"

	domainachievement "mcgame-server/internal/domain/achievement"
)

type BattleDerivedStateRepository interface {
	ListRequirementProgress(ctx context.Context, characterID int64) ([]*domainachievement.RequirementProgress, error)
}

type BattleDerivedStateReader struct {
	repo BattleDerivedStateRepository
}

func NewBattleDerivedStateReader(repo BattleDerivedStateRepository) *BattleDerivedStateReader {
	return &BattleDerivedStateReader{repo: repo}
}

func (r *BattleDerivedStateReader) Name() string {
	return "battle-derived"
}

func (r *BattleDerivedStateReader) RequirementIDs() []int64 {
	return []int64{324, 325}
}

func (r *BattleDerivedStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("battle-derived state repository not configured")
	}
	rows, err := r.repo.ListRequirementProgress(ctx, characterID)
	if err != nil {
		return nil, err
	}
	done := make(map[int64]bool, len(rows))
	for _, row := range rows {
		if row != nil && row.Done {
			done[row.RequirementID] = true
		}
	}
	return []StateProof{
		stateProof(324, boolStateProgress(allRequirementsDone(done, 194, 195, 196, 197)), 1),
		stateProof(325, boolStateProgress(allRequirementsDone(done, 202, 203, 204, 205)), 1),
	}, nil
}

func allRequirementsDone(done map[int64]bool, requirementIDs ...int64) bool {
	for _, requirementID := range requirementIDs {
		if !done[requirementID] {
			return false
		}
	}
	return true
}
