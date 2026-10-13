package warsprite

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appwarsprite "mcgame-server/internal/application/warsprite"
	domainfeature "mcgame-server/internal/domain/statfeature"
	domainwsp "mcgame-server/internal/domain/warsprite"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type panelStateRepo struct {
	domainfeature.Repository
	states []*domainfeature.CharacterFeatureState
	err    error
	writes int
}

func (r *panelStateRepo) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	return r.states, r.err
}

func (r *panelStateRepo) UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error {
	r.writes++
	return nil
}

func TestGetSpWarDataReturnsPanelStateWithoutLoginPushOrWrite(t *testing.T) {
	existing := domainwsp.Default()
	existing.WObj[1] = 1105
	for _, tc := range []struct {
		name string
		cid  string
		repo *panelStateRepo
		want map[string]any
	}{
		{"existing", "1", &panelStateRepo{states: []*domainfeature.CharacterFeatureState{{FeatureKey: domainfeature.FeatureWarSprite, State: existing.Encode()}}}, existing.ToClientPayload()},
		{"fresh", "1", &panelStateRepo{}, domainwsp.Default().ToClientPayload()},
		{"load error", "1", &panelStateRepo{err: errors.New("unavailable")}, domainwsp.Default().ToClientPayload()},
		{"no character", "", &panelStateRepo{}, domainwsp.Default().ToClientPayload()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := appwarsprite.NewService(tc.repo, nil, nil, zap.NewNop())
			h := NewHandler(svc, zap.NewNop())
			result, err := h.GetSpWarData(&rtmp.RPCContext{Context: context.Background(), CharacterID: tc.cid}, nil)
			if err != nil || !reflect.DeepEqual(result, tc.want) {
				t.Fatalf("result=%#v error=%v, want %#v", result, err, tc.want)
			}
			if tc.repo.writes != 0 {
				t.Fatal("panel read overwrote player state")
			}
		})
	}
}
