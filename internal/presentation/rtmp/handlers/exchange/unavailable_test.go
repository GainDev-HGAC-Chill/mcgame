package exchange

import (
	"context"
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestExchangeUnavailableDoesNotTouchAccountOrReportCredit(t *testing.T) {
	h := NewHandler(nil, zap.NewNop())
	ctx := &rtmp.RPCContext{Context: context.Background(), AccountID: uuid.NewString(), CharacterID: "1"}
	result, err := h.ExchangeAwardToGoldBind(ctx, []interface{}{float64(1000)})
	if err != nil || result != nil {
		t.Fatalf("unexpected fabricated exchange result: %#v %v", result, err)
	}
	if _, err := h.ExchangeAwardToGoldBind(ctx, []interface{}{float64(-1)}); err == nil {
		t.Fatal("accepted negative exchange")
	}
	dispatcher := rtmp.NewRPCDispatcher(zap.NewNop())
	h.RegisterHandlers(dispatcher)
	conn := rtmp.NewConnection(1, nil, dispatcher, zap.NewNop())
	if _, err := dispatcher.Dispatch(conn, "exchangeAwardToGoldBind", []interface{}{float64(1000)}); err == nil {
		t.Fatal("accepted unauthenticated exchange")
	}
	if dispatcher.IsQuiet("exchangeAwardToGoldBind") {
		t.Fatal("new method must stay logged")
	}
}
