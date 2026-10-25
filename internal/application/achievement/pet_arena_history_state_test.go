// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
)

type petArenaHistoryStateRepositoryFixture struct {
	totalWins           int
	fifteenWinDayStreak int
	twentyWinDayStreak  int
	err                 error
}

func (r petArenaHistoryStateRepositoryFixture) GetPetArenaAchievementState(context.Context, int64) (int, int, int, error) {
	return r.totalWins, r.fifteenWinDayStreak, r.twentyWinDayStreak, r.err
}

func TestPetArenaHistoryStateReaderUsesDurableAggregate(t *testing.T) {
	reader := NewPetArenaHistoryStateReader(petArenaHistoryStateRepositoryFixture{
		totalWins: 800, fifteenWinDayStreak: 6, twentyWinDayStreak: 5,
	})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	done := make(map[int64]bool, len(proofs))
	for _, proof := range proofs {
		done[proof.RequirementID] = proof.Done
	}
	for _, requirementID := range []int64{411, 412, 413, 415, 416} {
		if !done[requirementID] {
			t.Fatalf("requirement %d should be done: %#v", requirementID, proofs)
		}
	}
	for _, requirementID := range []int64{414, 417} {
		if done[requirementID] {
			t.Fatalf("requirement %d should be incomplete: %#v", requirementID, proofs)
		}
	}
}

func TestPetArenaHistoryStateReaderRejectsInconsistentStreaks(t *testing.T) {
	reader := NewPetArenaHistoryStateReader(petArenaHistoryStateRepositoryFixture{fifteenWinDayStreak: 4, twentyWinDayStreak: 5})
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want inconsistent streak rejection")
	}
}
