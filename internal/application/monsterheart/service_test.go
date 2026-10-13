package monsterheart

import (
	"context"
	"testing"

	domainfeature "mcgame-server/internal/domain/statfeature"

	"go.uber.org/zap"
)

type panelReadRepository struct {
	domainfeature.Repository
	state  *domainfeature.CharacterFeatureState
	writes int
}

func (r *panelReadRepository) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	return []*domainfeature.CharacterFeatureState{r.state}, nil
}

func (r *panelReadRepository) UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error {
	r.writes++
	return nil
}

func TestLoadNormalizesLegacyBoxLevelWithoutWritingOrLosingState(t *testing.T) {
	levels := map[string]interface{}{"1": float64(0), "2": float64(4)}
	repo := &panelReadRepository{state: &domainfeature.CharacterFeatureState{
		CharacterID: 5,
		FeatureKey:  domainfeature.FeatureMonsterHeart,
		State: map[string]interface{}{
			"bag": map[string]interface{}{
				"ren": map[string]interface{}{
					"5001": map[string]interface{}{"itemId": float64(5001), "n": float64(2)},
				},
			},
			"data": map[string]interface{}{
				"heartBoxLev": levels,
				"hadActHole":  map[string]interface{}{"2": map[string]interface{}{"1": float64(5001)}},
			},
		},
	}}
	state, err := NewService(repo, zap.NewNop()).Load(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if state.HeartBoxLev[1] != 1 || state.HeartBoxLev[2] != 4 {
		t.Fatalf("loaded levels = %#v", state.HeartBoxLev)
	}
	if state.BagItemCount(5001) != 2 || state.GetHoleItem(2, 1) != 5001 {
		t.Fatal("panel load lost persisted bag or slot state")
	}
	if repo.writes != 0 || levels["1"] != float64(0) {
		t.Fatal("panel load must not repair persisted state or mutate repository input")
	}
}
