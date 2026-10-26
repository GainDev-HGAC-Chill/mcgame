// Open-sourced by BaoLT

// Magic-estate state reader proves estate-level and unlocked-land milestones.
package achievement

import (
	"context"
	"fmt"

	appfarm "mcgame-server/internal/application/farm"
	domainfarm "mcgame-server/internal/domain/farm"
)

type MagicEstateStateRepository interface {
	FindByCharacter(ctx context.Context, characterID int64) (*domainfarm.MagicEstateProfile, error)
	GetMagicEstateStealAchievementState(ctx context.Context, characterID int64) (maxDayStreak int, maxDailyWins int, err error)
}

type MagicEstateStateReader struct {
	repo MagicEstateStateRepository
}

func NewMagicEstateStateReader(repo MagicEstateStateRepository) *MagicEstateStateReader {
	return &MagicEstateStateReader{repo: repo}
}

func (r *MagicEstateStateReader) Name() string {
	return "magic-estate"
}

func (r *MagicEstateStateReader) RequirementIDs() []int64 {
	return append(requirementIDRange(399, 403), requirementIDRange(406, 410)...)
}

func (r *MagicEstateStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("magic-estate state repository not configured")
	}
	profile, err := r.repo.FindByCharacter(ctx, characterID)
	if err != nil {
		return nil, err
	}
	estateLevel := 0
	farmNum := 0
	if profile != nil {
		if profile.CharacterID != characterID {
			return nil, fmt.Errorf("magic-estate profile character %d does not match %d", profile.CharacterID, characterID)
		}
		estateLevel = appfarm.MagicEstateLevelByExp(profile.Exp)
		farmNum = profile.FarmNum
	}
	maxDayStreak, maxDailyWins, err := r.repo.GetMagicEstateStealAchievementState(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if maxDayStreak < 0 || maxDailyWins < 0 {
		return nil, fmt.Errorf("invalid magic-estate steal state for character %d", characterID)
	}
	return []StateProof{
		stateProof(399, estateLevel, 2),
		stateProof(400, estateLevel, 5),
		stateProof(401, farmNum, 4),
		stateProof(402, farmNum, 8),
		stateProof(403, farmNum, 16),
		stateProof(406, maxDayStreak, 7),
		stateProof(407, maxDayStreak, 30),
		stateProof(408, maxDayStreak, 90),
		stateProof(409, maxDailyWins, 30),
		stateProof(410, maxDailyWins, 60),
	}, nil
}
