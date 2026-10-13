package prs

import (
	"context"
	"reflect"
	"testing"

	appprs "mcgame-server/internal/application/prs"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/infrastructure/rtmp"
	"mcgame-server/internal/presentation/rtmp/handlers/activity"
	rtmpmsg "mcgame-server/pkg/rtmp/message"

	"go.uber.org/zap"
)

type registrationTransport struct {
	panelTransport
	transactionID int64
	result        interface{}
	faults        int
}

func (s *registrationTransport) WriteResult(_ context.Context, id int64, result interface{}) error {
	s.transactionID = id
	s.result = result
	return nil
}

func (s *registrationTransport) WriteError(context.Context, int64, error) error {
	s.faults++
	return nil
}

func TestStartupRegistrationKeepsPersistedPRSResponderAfterActivity(t *testing.T) {
	repo := &panelRepository{states: []*domainfeature.CharacterFeatureState{{
		FeatureKey: domainfeature.FeaturePRS,
		State: map[string]interface{}{
			"tid": "3", "useSid": float64(2),
			"actArr":      map[string]interface{}{"0": float64(2)},
			"actLimitObj": map[string]interface{}{"11": float64(2000000000000)},
			"chipBag":     map[string]interface{}{"1": map[string]interface{}{"chipId": float64(4), "chipNum": float64(7)}},
			"excBag":      map[string]interface{}{}, "isShow": float64(1),
		},
	}}}
	logger := zap.NewNop()
	dispatcher := rtmp.NewRPCDispatcher(logger)
	NewHandler(appprs.NewService(repo, nil, nil, logger), logger).RegisterHandlers(dispatcher)
	activity.NewHandler(nil, logger).RegisterHandlers(dispatcher)
	sender := &registrationTransport{}
	conn := rtmp.NewConnection(1, nil, dispatcher, logger)
	conn.SetTransport(sender)
	conn.SetCharacter("3")
	err := conn.OnUnknownCommandMessage(0, &rtmpmsg.CommandMessage{
		CommandName: "initPRSPanel", TransactionID: 17,
		DecodedArgs: []interface{}{nil, float64(3)},
	})
	if err != nil {
		t.Fatal(err)
	}
	panel := assertPanelContract(t, sender.result)
	want := map[string]interface{}{
		"tid": "3", "useSid": 2, "actArr": map[string]interface{}{"0": 2},
		"actLimitObj": map[string]interface{}{"11": int64(2000000000000)},
		"chipBag":     map[string]interface{}{"1": map[string]interface{}{"chipId": 4, "chipNum": 7}},
		"excBag":      map[string]interface{}{}, "isShow": 1,
	}
	if !reflect.DeepEqual(panel, want) {
		t.Fatalf("combined registration lost persisted PRS state: got %#v, want %#v", panel, want)
	}
	if sender.transactionID != 17 || sender.faults != 0 || len(sender.methods) != 0 || repo.characterID != 3 || repo.writes != 0 {
		t.Fatalf("expected only persisted-service Responder result: sender=%#v repo=%#v", sender, repo)
	}
}
