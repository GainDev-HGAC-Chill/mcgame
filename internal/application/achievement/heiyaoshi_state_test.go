// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	appheiyaoshi "mcgame-server/internal/application/heiyaoshi"
	domainfeature "mcgame-server/internal/domain/statfeature"
)

type characterFeatureStateRepositoryFixture struct {
	states []*domainfeature.CharacterFeatureState
}

func (r characterFeatureStateRepositoryFixture) ListCharacterFeatureStates(ctx context.Context, characterID int64) ([]*domainfeature.CharacterFeatureState, error) {
	return r.states, nil
}

func TestHeiyaoshiStateReaderUsesCompletedStageOneFigures(t *testing.T) {
	state := appheiyaoshi.NewState()
	state.LastActFigure = 7
	repo := characterFeatureStateRepositoryFixture{states: []*domainfeature.CharacterFeatureState{{
		CharacterID: 99,
		FeatureKey:  domainfeature.FeatureHeiyaoshi,
		State:       state.Encode(),
	}}}
	proofs, err := NewHeiyaoshiStateReader(repo).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 11 {
		t.Fatalf("proof count = %d, want 11", len(proofs))
	}
	for index, proof := range proofs {
		wantDone := index < 6
		if proof.Done != wantDone || proof.Progress != 6 {
			t.Fatalf("proof %d = %#v, want done=%t progress=6", index, proof, wantDone)
		}
	}
}

var _ CharacterFeatureStateRepository = characterFeatureStateRepositoryFixture{}
