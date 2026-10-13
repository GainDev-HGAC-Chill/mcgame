package activity

import (
	"context"
	"reflect"
	"testing"
	"time"

	appactivity "mcgame-server/internal/application/activity"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type pmResponderTransport struct {
	methods []string
	args    [][]interface{}
}

func (s *pmResponderTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	s.methods = append(s.methods, method)
	s.args = append(s.args, args)
	return nil
}

func (s *pmResponderTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return s.WriteCommand(ctx, method, args)
}

func (s *pmResponderTransport) WriteResult(context.Context, int64, interface{}) error { return nil }
func (s *pmResponderTransport) WriteError(context.Context, int64, error) error        { return nil }
func (s *pmResponderTransport) Kind() string                                          { return "test" }
func (s *pmResponderTransport) Close() error                                          { return nil }

func newPMResponderHandler(gold int64) (*Handler, *rtmp.RPCContext, *pmResponderTransport, *premiumHandlerCharacterRepo) {
	logger := zap.NewNop()
	repo := &premiumHandlerCharacterRepo{char: &domainchar.Character{ID: 101, Gold: gold}}
	service := appactivity.NewPremiumService(repo, nil, logger)
	service.SetNowFunc(func() time.Time { return time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC) })
	handler := NewHandler(nil, logger)
	handler.SetPremiumService(service)
	sender := &pmResponderTransport{}
	conn := rtmp.NewConnection(1, nil, nil, logger)
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{Context: context.Background(), ConnID: 1, CharacterID: "101", Connection: conn}
	return handler, ctx, sender, repo
}

func TestInitPmDataReturnsResponderPayloadWithoutNamedCallback(t *testing.T) {
	handler, ctx, sender, _ := newPMResponderHandler(1000)
	result, err := handler.InitPmData(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC).UnixMilli()
	want := map[string]interface{}{
		"data": map[string]interface{}{
			"pmExp": int64(0), "activeType": 0, "keepDay": float64(0), "activeTime": now,
		},
		"date": now, "flag": 0, "processFlag": map[string]interface{}{}, "findback": false,
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("responder payload = %#v, want %#v", result, want)
	}
	if len(sender.methods) != 0 {
		t.Fatalf("initPmData sent named callbacks: %v", sender.methods)
	}
	ctx.Connection = nil
	if result, err := handler.InitPmData(ctx, nil); err != nil || result == nil {
		t.Fatalf("without connection result = %#v, error = %v", result, err)
	}
}

func TestBuyPmReturnsResponderPayloadAndRetainsValidPushes(t *testing.T) {
	handler, ctx, sender, repo := newPMResponderHandler(1000)
	result, err := handler.BuyPm(ctx, []interface{}{float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC).UnixMilli()
	want := map[string]interface{}{
		"level": 1,
		"data": map[string]interface{}{
			"pmExp": int64(688), "activeType": 1, "keepDay": float64(30), "activeTime": now,
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("responder payload = %#v, want %#v", result, want)
	}
	wantMethods := []string{"onAddPm", "onAddMoney", "onUPP", "updateScenePmLevel"}
	if !reflect.DeepEqual(sender.methods, wantMethods) {
		t.Fatalf("named callbacks = %v, want %v", sender.methods, wantMethods)
	}
	wantArgs := [][]interface{}{
		{want},
		{float64(101), "gold", float64(-688), float64(312)},
		{map[string]interface{}{"gold": int64(312)}},
		{map[string]interface{}{"cid": int64(101), "pmLevel": 1}},
	}
	if !reflect.DeepEqual(sender.args, wantArgs) {
		t.Fatalf("callback arguments = %#v, want %#v", sender.args, wantArgs)
	}
	if repo.char.Gold != 312 || repo.char.VIPType != 1 {
		t.Fatalf("purchase state = %#v", repo.char)
	}
}

func TestBuyPmFailureReturnsFalsyResponderAndOnlyErrorPush(t *testing.T) {
	for _, missingCharacter := range []bool{false, true} {
		handler, ctx, sender, repo := newPMResponderHandler(100)
		if missingCharacter {
			repo.char = nil
		}
		result, err := handler.BuyPm(ctx, []interface{}{float64(1)})
		if err != nil || result != false {
			t.Fatalf("failure result = %#v, error = %v", result, err)
		}
		if !reflect.DeepEqual(sender.methods, []string{"onRedMsg"}) {
			t.Fatalf("failure callbacks = %v", sender.methods)
		}
		if repo.char != nil && (repo.char.Gold != 100 || repo.char.VIPType != 0) {
			t.Fatalf("failure mutated membership or gold: %#v", repo.char)
		}
	}
}

func TestPMResponderValidationReturnsRPCErrorWithoutPMCallbacks(t *testing.T) {
	for _, test := range []struct {
		name string
		init bool
		char string
		args []interface{}
	}{
		{name: "init unauthenticated", init: true},
		{name: "init invalid character", init: true, char: "invalid"},
		{name: "buy unauthenticated", args: []interface{}{float64(1)}},
		{name: "buy missing package", char: "101"},
		{name: "buy invalid type", char: "101", args: []interface{}{true}},
		{name: "buy invalid range", char: "101", args: []interface{}{float64(4)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, ctx, sender, _ := newPMResponderHandler(1000)
			ctx.CharacterID = test.char
			fn := handler.BuyPm
			if test.init {
				fn = handler.InitPmData
			}
			if result, err := fn(ctx, test.args); err == nil || result != nil {
				t.Fatalf("result = %#v, error = %v, want RPC error", result, err)
			}
			if len(sender.methods) != 0 {
				t.Fatalf("validation callbacks = %v", sender.methods)
			}
		})
	}
}
