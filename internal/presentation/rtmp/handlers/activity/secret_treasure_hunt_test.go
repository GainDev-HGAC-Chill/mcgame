package activity

import (
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func TestSecretTreasureHuntClosedGateResponderContract(t *testing.T) {
	h := NewHandler(nil, zap.NewNop())
	dispatcher := rtmp.NewRPCDispatcher(zap.NewNop())
	h.RegisterHandlers(dispatcher)
	conn := rtmp.NewConnection(1, nil, dispatcher, zap.NewNop())
	conn.SetCharacter("1")
	result, err := dispatcher.Dispatch(conn, "sthPlayIsInGroup", nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("responder payload = %T, want object with openflag and group", result)
	}
	if len(payload) != 2 || payload["openflag"] != false || payload["group"] != false {
		t.Fatalf("unconfigured event must stay closed without invented state: %#v", payload)
	}
}
