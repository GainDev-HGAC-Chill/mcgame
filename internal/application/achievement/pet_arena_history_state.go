// Open-sourced by BaoLT

// Pet-arena history state reader proves durable wins and qualifying-day streaks.
package achievement

import (
	"context"
	"fmt"
)

type PetArenaHistoryStateRepository interface {
	GetPetArenaAchievementState(ctx context.Context, characterID int64) (totalWins int, fifteenWinDayStreak int, twentyWinDayStreak int, err error)
}

type PetArenaHistoryStateReader struct {
	repo PetArenaHistoryStateRepository
}

func NewPetArenaHistoryStateReader(repo PetArenaHistoryStateRepository) *PetArenaHistoryStateReader {
	return &PetArenaHistoryStateReader{repo: repo}
}

func (r *PetArenaHistoryStateReader) Name() string {
	return "pet-arena-history"
}

func (r *PetArenaHistoryStateReader) RequirementIDs() []int64 {
	return []int64{411, 412, 413, 414, 415, 416, 417}
}

func (r *PetArenaHistoryStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("pet-arena-history state repository not configured")
	}
	totalWins, fifteenWinDayStreak, twentyWinDayStreak, err := r.repo.GetPetArenaAchievementState(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if totalWins < 0 || fifteenWinDayStreak < 0 || twentyWinDayStreak < 0 || twentyWinDayStreak > fifteenWinDayStreak {
		return nil, fmt.Errorf("invalid pet-arena history state for character %d", characterID)
	}
	return []StateProof{
		stateProof(411, totalWins, 1),
		stateProof(412, totalWins, 70),
		stateProof(413, totalWins, 700),
		stateProof(414, totalWins, 7000),
		stateProof(415, fifteenWinDayStreak, 3),
		stateProof(416, twentyWinDayStreak, 5),
		stateProof(417, twentyWinDayStreak, 7),
	}, nil
}
