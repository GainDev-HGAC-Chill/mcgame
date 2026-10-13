package rtmp

import (
	"context"
	"testing"

	pkgerrors "mcgame-server/pkg/errors"
	rtmpmsg "mcgame-server/pkg/rtmp/message"

	"go.uber.org/zap"
)

type finishQuestResultTransport struct {
	results map[int64]interface{}
	errors  int
}

func (s *finishQuestResultTransport) WriteCommand(context.Context, string, []interface{}) error {
	return nil
}
func (s *finishQuestResultTransport) WriteCommandSync(context.Context, string, []interface{}) error {
	return nil
}
func (s *finishQuestResultTransport) WriteResult(_ context.Context, id int64, result interface{}) error {
	s.results[id] = result
	return nil
}
func (s *finishQuestResultTransport) WriteError(context.Context, int64, error) error {
	s.errors++
	return nil
}
func (s *finishQuestResultTransport) Kind() string { return "test" }
func (s *finishQuestResultTransport) Close() error { return nil }

func TestAwaitedFinishQuestRateLimitReturnsTerminalResult(t *testing.T) {
	logger := zap.NewNop()
	dispatcher := NewRPCDispatcher(logger)
	calls := 0
	dispatcher.Register("finishQuest", func(*RPCContext, []interface{}) (interface{}, error) {
		calls++
		return map[string]interface{}{"flag": true}, nil
	})
	conn := NewConnection(1, nil, dispatcher, logger)
	transport := &finishQuestResultTransport{results: make(map[int64]interface{})}
	conn.SetTransport(transport)
	for _, id := range []int64{1, 2} {
		if err := conn.OnUnknownCommandMessage(0, &rtmpmsg.CommandMessage{CommandName: "finishQuest", TransactionID: id, DecodedArgs: []interface{}{nil, float64(6)}}); err != nil {
			t.Fatal(err)
		}
	}
	result, ok := transport.results[2].(map[string]interface{})
	if calls != 1 || !ok || result["flag"] != false || result["info"] != pkgerrors.ErrRateLimited.Error() || transport.errors != 0 {
		t.Fatalf("second request stranded/replayed: calls=%d results=%v errors=%d", calls, transport.results, transport.errors)
	}
}

func TestAwaitedFinishQuestHandlerErrorReturnsResultNotFault(t *testing.T) {
	for _, failure := range []error{pkgerrors.ErrInvalidArgs, pkgerrors.ErrUnauthorized} {
		logger := zap.NewNop()
		dispatcher := NewRPCDispatcher(logger)
		dispatcher.Register("finishQuest", func(*RPCContext, []interface{}) (interface{}, error) { return nil, failure })
		conn := NewConnection(1, nil, dispatcher, logger)
		transport := &finishQuestResultTransport{results: make(map[int64]interface{})}
		conn.SetTransport(transport)
		if err := conn.OnUnknownCommandMessage(0, &rtmpmsg.CommandMessage{CommandName: "finishQuest", TransactionID: 1, DecodedArgs: []interface{}{nil}}); err != nil {
			t.Fatal(err)
		}
		result, ok := transport.results[1].(map[string]interface{})
		if !ok || result["flag"] != false || result["info"] != failure.Error() || transport.errors != 0 {
			t.Fatalf("missing responder failure: results=%v errors=%d", transport.results, transport.errors)
		}
	}
}

func TestAwaitedFurnaceDispatchErrorReturnsFalsyResult(t *testing.T) {
	for _, method := range []string{"MysteryExchangeItem", "MysteryExchangeScore"} {
		logger := zap.NewNop()
		dispatcher := NewRPCDispatcher(logger)
		dispatcher.Register(method, func(*RPCContext, []interface{}) (interface{}, error) { return nil, pkgerrors.ErrRateLimited })
		conn := NewConnection(1, nil, dispatcher, logger)
		transport := &finishQuestResultTransport{results: make(map[int64]interface{})}
		conn.SetTransport(transport)
		if err := conn.OnUnknownCommandMessage(0, &rtmpmsg.CommandMessage{CommandName: method, TransactionID: 1, DecodedArgs: []interface{}{nil}}); err != nil {
			t.Fatal(err)
		}
		if transport.results[1] != false || transport.errors != 0 {
			t.Fatalf("%s did not release Flash waiter: %+v", method, transport)
		}
	}
}
