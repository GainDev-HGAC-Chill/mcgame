package dailysignin

import (
	"context"
	"errors"
	"testing"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/dailysignin"
	pkgerrors "mcgame-server/pkg/errors"
)

func TestPaidSignInWalletTotals(t *testing.T) {
	for _, action := range []string{"retroactive", "crit"} {
		for _, wallet := range []struct {
			name                             string
			gold, bound, wantGold, wantBound int64
		}{
			{"bound only", 20, 300, 20, 200},
			{"mixed", 200, 40, 140, 0},
			{"unbound only", 200, 0, 100, 0},
		} {
			t.Run(action+"/"+wallet.name, func(t *testing.T) {
				char := &domainchar.Character{ID: 1, Level: 50, Gold: wallet.gold, GoldBind: wallet.bound, Money: 777, GenericPoint: 888}
				svc, _, chars, _ := newTestService(&dailysignin.Record{ConsumeLimitTotal: 499}, nil, nil, char, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC), 1)
				cfg := svc.GetConfig()
				cfg.RetroactivePrice = 100
				svc.SetConfig(cfg)
				call := svc.UpCrit
				if action == "retroactive" {
					call = svc.Retroactive
				}
				out, err := call(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
				if chars.char.Gold != wallet.wantGold || chars.char.GoldBind != wallet.wantBound {
					t.Fatalf("persisted wallet = %d/%d, want %d/%d", chars.char.Gold, chars.char.GoldBind, wallet.wantGold, wallet.wantBound)
				}
				if out.GoldSpent != 100 || out.GoldLeft != wallet.wantGold || out.GoldBindLeft != wallet.wantBound {
					t.Fatalf("incorrect debit outcome: %+v", out)
				}
				if out.Record.ConsumeLimitTotal != 599 {
					t.Fatalf("cumulative spend = %d, want 599", out.Record.ConsumeLimitTotal)
				}
				if chars.char.Money != 777 || chars.char.GenericPoint != 888 {
					t.Fatal("non-Gold wallet was changed")
				}
			})
		}
	}
}

func TestPaidSignInInsufficientFundsDoNotMutate(t *testing.T) {
	for _, action := range []string{"retroactive", "crit"} {
		t.Run(action, func(t *testing.T) {
			char := &domainchar.Character{ID: 1, Level: 50, Gold: 10, GoldBind: 20}
			svc, granter, chars, records := newTestService(&dailysignin.Record{ConsumeLimitTotal: 499}, defaultRewards(), nil, char, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC), 1)
			call := svc.UpCrit
			if action == "retroactive" {
				call = svc.Retroactive
			}
			out, err := call(context.Background(), 1)
			if !errors.Is(err, pkgerrors.ErrInsufficientFunds) || out != nil {
				t.Fatalf("out=%+v err=%v", out, err)
			}
			if chars.char.Gold != 10 || chars.char.GoldBind != 20 || records.rec.ConsumeLimitTotal != 499 || records.rec.CritPercent != 0 || records.rec.ClaimedDaysBitmap != 0 || len(granter.granted) != 0 {
				t.Fatal("rejected purchase mutated wallet, ledger, or rewards")
			}
		})
	}
}

func TestLuckyGoldCreditKeepsBoundWalletSeparate(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	rec := &dailysignin.Record{ConsumeLimitTotal: 600}
	for day := 1; day <= now.Day(); day++ {
		rec.MarkClaimed(day)
	}
	tiers := map[int]dailysignin.LuckyTier{1: {Year: 2026, Month: 5, WeekIndex: 1, Threshold: 499, GoldAmount: 500}}
	svc, _, chars, _ := newTestService(rec, nil, tiers, &domainchar.Character{ID: 1, Level: 50, Gold: 20, GoldBind: 300}, now, 1)
	out, err := svc.GetAward(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if out.GoldSpent != -500 || out.GoldLeft != 520 || out.GoldBindLeft != 300 || chars.char.Gold != 520 || chars.char.GoldBind != 300 || out.Record.ConsumeLimitTotal != 600 {
		t.Fatalf("incorrect credit outcome: %+v wallet=%+v", out, chars.char)
	}
}
