package activity

import (
	"context"
	"strconv"
	"testing"
	"time"

	apppet "mcgame-server/internal/application/pet"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type ppvePanelContractRepository struct {
	domainfeature.Repository
	state  *domainfeature.CharacterFeatureState
	writes int
}

func (r *ppvePanelContractRepository) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	return []*domainfeature.CharacterFeatureState{r.state}, nil
}

func (r *ppvePanelContractRepository) UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error {
	r.writes++
	return nil
}

func TestInitPPVEPanelPushContractProvidesEightPartsWithoutGrants(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	repo := &ppvePanelContractRepository{state: &domainfeature.CharacterFeatureState{
		FeatureKey: domainfeature.FeaturePetPVE,
		State: map[string]interface{}{
			"lastResetDay": today, "lastDailyDay": today,
			"p": map[string]interface{}{}, "mlv": 0, "kp": 0,
		},
	}}
	logger := zap.NewNop()
	h := NewHandler(nil, logger)
	h.SetPPVEService(apppet.NewPPVEService(repo, nil, logger))
	sender := &pmResponderTransport{}
	conn := rtmp.NewConnection(1, nil, nil, logger)
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{Context: context.Background(), ConnID: 1, CharacterID: "101", Connection: conn}
	result, err := h.InitPPVEPanel(ctx, []interface{}{nil})
	if err != nil || result != nil {
		t.Fatalf("null-Responder result = %#v, err = %v", result, err)
	}
	if len(sender.methods) != 1 || sender.methods[0] != "onUpdatePPVEPanel" || len(sender.args[0]) != 4 {
		t.Fatalf("push contract: methods=%v args=%#v", sender.methods, sender.args)
	}
	args := sender.args[0]
	kp := args[0].(map[string]interface{})
	data := args[1].(map[string]interface{})
	if kp["kp"] != 0 || kp["ppvefloor"] != 0 || data["mlv"] != 0 || args[2] != nil || args[3] != false {
		t.Fatalf("push arguments = %#v", args)
	}
	parts := data["p"].(map[string]interface{})
	for index := 0; index < 8; index++ {
		part, ok := parts["s"+strconv.Itoa(index)].(map[string]interface{})
		if !ok || part == nil || len(part) != 0 {
			t.Fatalf("carve part %d = %#v", index, part)
		}
	}
	if repo.writes != 0 || len(repo.state.State["p"].(map[string]interface{})) != 0 {
		t.Fatalf("protocol-only normalization persisted state: %#v", repo)
	}
}
