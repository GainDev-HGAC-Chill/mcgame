// Open-sourced by BaoLT

// Marriage-history state reader proves persisted wedding-count and current-duration milestones.
package achievement

import (
	"context"
	"fmt"
)

type MarriageHistoryStateRepository interface {
	GetMarriageAchievementState(ctx context.Context, characterID int64) (normalWeddings int, marriageCount int, currentMarriedDays int, err error)
}

type MarriageHistoryStateReader struct {
	repo MarriageHistoryStateRepository
}

func NewMarriageHistoryStateReader(repo MarriageHistoryStateRepository) *MarriageHistoryStateReader {
	return &MarriageHistoryStateReader{repo: repo}
}

func (r *MarriageHistoryStateReader) Name() string {
	return "marriage-history"
}

func (r *MarriageHistoryStateReader) RequirementIDs() []int64 {
	return []int64{131, 346, 347, 356, 357, 358, 359}
}

func (r *MarriageHistoryStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("marriage-history state repository not configured")
	}
	normalWeddings, marriageCount, currentMarriedDays, err := r.repo.GetMarriageAchievementState(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if normalWeddings < 0 || marriageCount < 0 || currentMarriedDays < 0 || normalWeddings > marriageCount {
		return nil, fmt.Errorf("invalid marriage-history state for character %d", characterID)
	}
	return []StateProof{
		stateProof(131, normalWeddings, 1),
		stateProof(346, marriageCount, 3),
		stateProof(347, marriageCount, 5),
		stateProof(356, currentMarriedDays, 90),
		stateProof(357, currentMarriedDays, 180),
		stateProof(358, currentMarriedDays, 360),
		stateProof(359, currentMarriedDays, 720),
	}, nil
}
