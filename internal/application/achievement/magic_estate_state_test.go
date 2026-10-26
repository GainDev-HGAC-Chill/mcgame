package achievement

import (
	"context"
	"errors"
	"testing"

	domainfarm "mcgame-server/internal/domain/farm"
)

type magicEstateStateRepoFixture struct {
	profile      *domainfarm.MagicEstateProfile
	maxDayStreak int
	maxDailyWins int
	err          error
}

func (r magicEstateStateRepoFixture) GetMagicEstateStealAchievementState(context.Context, int64) (int, int, error) {
	return r.maxDayStreak, r.maxDailyWins, r.err
}

func (r magicEstateStateRepoFixture) FindByCharacter(context.Context, int64) (*domainfarm.MagicEstateProfile, error) {
	return r.profile, r.err
}

func TestMagicEstateStateReaderUsesEstateLevelAndUnlockedLand(t *testing.T) {
	reader := NewMagicEstateStateReader(magicEstateStateRepoFixture{profile: &domainfarm.MagicEstateProfile{
		CharacterID: 99,
		Exp:         61,
		FarmNum:     8,
	}, maxDayStreak: 30, maxDailyWins: 45})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 10 {
		t.Fatalf("proof count = %d, want 10", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{399, 401, 402, 406, 407, 409} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{400, 403, 408, 410} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestMagicEstateStateReaderTreatsMissingProfileAsNoProgress(t *testing.T) {
	reader := NewMagicEstateStateReader(magicEstateStateRepoFixture{})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	for _, proof := range proofs {
		if proof.Progress != 0 || proof.Done {
			t.Fatalf("proof = %#v, want zero incomplete", proof)
		}
	}
}

func TestMagicEstateStateReaderFailsClosed(t *testing.T) {
	repoErr := errors.New("read failed")
	reader := NewMagicEstateStateReader(magicEstateStateRepoFixture{err: repoErr})
	if _, err := reader.ReadStateProofs(context.Background(), 99); !errors.Is(err, repoErr) {
		t.Fatalf("ReadStateProofs() error = %v, want %v", err, repoErr)
	}
	reader = NewMagicEstateStateReader(magicEstateStateRepoFixture{profile: &domainfarm.MagicEstateProfile{CharacterID: 100}})
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want mismatched profile error")
	}
}
