package activity

import (
	"context"
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func TestUnconfiguredAwardReadsContainNoSyntheticRewards(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1"}
	for _, fn := range []rtmp.HandlerFunc{h.GetPointAward, h.GetShopAward, h.GetConsume} {
		result, err := fn(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		m, ok := result.(map[string]interface{})
		if !ok || m == nil || len(m) != 0 {
			t.Fatalf("expected empty reward dictionary: %#v", result)
		}
	}
	result, err := h.SerchForGift(ctx, nil)
	if err != nil || result != false {
		t.Fatalf("gift availability=%#v error=%v", result, err)
	}
	result, err = h.SearchGameGift(ctx, nil)
	if err != nil || result != "[]" {
		t.Fatalf("gift JSON=%#v error=%v", result, err)
	}
	result, err = h.GetStageConsumeAwardList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	stages := result.([]interface{})
	if stages[0].(map[string]interface{})["stageNum"] != 0 || len(stages[1].(map[string]interface{})) != 0 {
		t.Fatalf("invented stages/totals: %#v", result)
	}
	result, err = h.GetConsumeAwardList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	consume := result.([]interface{})
	if len(consume[0].(map[string]interface{})) != 0 || consume[1] != 0 {
		t.Fatalf("invented consume awards: %#v", result)
	}
	result, err = h.GetActAwardInfo(ctx, []interface{}{float64(0)})
	if err != nil || result != nil {
		t.Fatalf("fabricated new-server event: %#v %v", result, err)
	}
}

func TestUnconfiguredAwardReadsRequireCharacter(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	ctx := &rtmp.RPCContext{Context: context.Background()}
	for _, fn := range []rtmp.HandlerFunc{h.GetPointAward, h.GetShopAward, h.GetConsume, h.GetStageConsumeAwardList, h.GetConsumeAwardList, h.SearchGameGift, h.SerchForGift, h.GetActAwardInfo} {
		if _, err := fn(ctx, []interface{}{float64(0)}); err == nil {
			t.Fatal("accepted unauthenticated read")
		}
	}
}

func TestUnconfiguredGiftAndServerEventMethodsRegistered(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	dispatcher := rtmp.NewRPCDispatcher(zap.NewNop())
	h.RegisterHandlers(dispatcher)
	conn := rtmp.NewConnection(1, nil, dispatcher, zap.NewNop())
	conn.SetCharacter("1")
	for _, request := range []struct {
		method string
		args   []interface{}
	}{
		{"serchForGift", nil},
		{"getActAwardInfo", []interface{}{float64(0)}},
		{"getConsume", nil},
	} {
		if _, err := dispatcher.Dispatch(conn, request.method, request.args); err != nil {
			t.Fatalf("%s dispatch error: %v", request.method, err)
		}
		if dispatcher.IsQuiet(request.method) {
			t.Fatalf("new RPC %s must remain logged", request.method)
		}
	}
}
