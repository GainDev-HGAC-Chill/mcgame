// Open-sourced by BaoLT

// Final-stat state reader proves effective character defense and speed milestones.
package achievement

import (
	"context"
	"fmt"

	domaincharacter "mcgame-server/internal/domain/character"
)

type FinalStatsBonusProvider interface {
	AggregateEquipmentStats(ctx context.Context, characterID int64) domaincharacter.EquipmentStatBonuses
}

type FinalStatsStateReader struct {
	repo     CharacterStateRepository
	provider FinalStatsBonusProvider
}

func NewFinalStatsStateReader(repo CharacterStateRepository, provider FinalStatsBonusProvider) *FinalStatsStateReader {
	return &FinalStatsStateReader{repo: repo, provider: provider}
}

func (r *FinalStatsStateReader) Name() string {
	return "final-stats"
}

func (r *FinalStatsStateReader) RequirementIDs() []int64 {
	return []int64{47, 48, 49}
}

func (r *FinalStatsStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.provider == nil {
		return nil, fmt.Errorf("final-stats state dependencies not configured")
	}
	character, err := r.repo.FindByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if character == nil || character.ID != characterID {
		return nil, fmt.Errorf("final-stats character %d not found", characterID)
	}
	effective := *character
	effective.RecalculateStats()
	effective.ApplyEquipmentBonuses(r.provider.AggregateEquipmentStats(ctx, characterID))
	return []StateProof{
		stateProof(47, effective.Defense, 40000),
		stateProof(48, effective.MagicDefense, 40000),
		stateProof(49, effective.Speed, 15000),
	}, nil
}
