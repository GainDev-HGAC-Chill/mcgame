package dailysignin

import (
	"context"
	"testing"

	appdailysignin "mcgame-server/internal/application/dailysignin"
	"mcgame-server/internal/domain/dailysignin"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type walletCallback struct {
	method string
	args   []interface{}
}

type walletTransport struct {
	calls []walletCallback
}

func (s *walletTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	s.calls = append(s.calls, walletCallback{method: method, args: args})
	return nil
}

func (s *walletTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return s.WriteCommand(ctx, method, args)
}

func (s *walletTransport) WriteResult(context.Context, int64, interface{}) error { return nil }
func (s *walletTransport) WriteError(context.Context, int64, error) error        { return nil }
func (s *walletTransport) Kind() string                                          { return "test" }
func (s *walletTransport) Close() error                                          { return nil }

func TestWalletCallbacksPublishSeparateAbsoluteTotals(t *testing.T) {
	for _, tc := range []struct {
		name                                        string
		spent, gold, bound, beforeGold, beforeBound int64
	}{
		{"bound debit", 100, 20, 200, 20, 300},
		{"mixed debit", 100, 140, 0, 200, 40},
		{"unbound debit", 100, 100, 0, 200, 0},
		{"lucky credit", -500, 520, 300, 20, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := zap.NewNop()
			h := NewHandler(nil, logger)
			sender := &walletTransport{}
			conn := rtmp.NewConnection(1, nil, nil, logger)
			conn.SetTransport(sender)
			ctx := &rtmp.RPCContext{Connection: conn}
			out := &appdailysignin.Outcome{GoldSpent: tc.spent, GoldLeft: tc.gold, GoldBindLeft: tc.bound, Record: &dailysignin.Record{ConsumeLimitTotal: 599}}
			h.pushGoldDebit(ctx, out)
			h.pushGoldCredit(ctx, out)
			h.pushConsumeLimit(ctx, out)
			if len(sender.calls) != 2 || sender.calls[0].method != "onUPP" || sender.calls[1].method != "oninitDailySignInActConsumeLimit" {
				t.Fatalf("callbacks = %+v", sender.calls)
			}
			payload := sender.calls[0].args[0].(map[string]interface{})
			if len(payload) != 2 || payload["gold"] != tc.gold || payload["goldBind"] != tc.bound {
				t.Fatalf("wallet payload = %#v", payload)
			}
			client := map[string]int64{"gold": tc.beforeGold, "goldBind": tc.beforeBound}
			for key, value := range payload {
				if _, exists := client[key]; exists {
					client[key] = value.(int64)
				}
			}
			if delta := client["gold"] + client["goldBind"] - tc.beforeGold - tc.beforeBound; delta != -tc.spent {
				t.Fatalf("Flash wallet delta = %d, want %d", delta, -tc.spent)
			}
			if sender.calls[1].args[0] != int64(599) {
				t.Fatalf("spend callback = %#v", sender.calls[1].args)
			}
		})
	}
}

func TestZeroSpendHasNoWalletCallback(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandler(nil, logger)
	sender := &walletTransport{}
	conn := rtmp.NewConnection(1, nil, nil, logger)
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{Connection: conn}
	out := &appdailysignin.Outcome{GoldLeft: 20, GoldBindLeft: 300}
	h.pushGoldDebit(ctx, out)
	h.pushGoldCredit(ctx, out)
	if len(sender.calls) != 0 {
		t.Fatalf("unexpected wallet callbacks: %+v", sender.calls)
	}
}
