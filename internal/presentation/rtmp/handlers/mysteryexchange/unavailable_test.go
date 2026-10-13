package mysteryexchange

import (
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func TestRegisteredFurnaceExchangesFailClosedBeforeAnyDependency(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandler(nil, nil, logger)
	dispatcher := rtmp.NewRPCDispatcher(logger)
	h.RegisterHandlers(dispatcher)
	conn := rtmp.NewConnection(1, nil, dispatcher, logger)
	for _, method := range []string{"MysteryExchangeItem", "MysteryExchangeScore"} {
		result, err := dispatcher.Dispatch(conn, method, []interface{}{map[string]interface{}{"1": true}, "not-a-password"})
		if err != nil || result != false {
			t.Fatalf("%s reached unsafe legacy exchange: result=%v err=%v", method, result, err)
		}
	}
}
