package activity

import (
	"context"
	"errors"
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func TestGetJXHDAwardEmptyPreviewContract(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1"}
	for _, key := range []string{"ConsumeAward", "Wawagame", "PointAward", "ComboShop", "Juhuasuan", "Treasurebowl", "Happyline", "PointNotice"} {
		t.Run(key, func(t *testing.T) {
			result, err := h.GetJXHDAward(ctx, []interface{}{key})
			if err != nil {
				t.Fatal(err)
			}
			array, ok := result.([]interface{})
			if !ok || len(array) != 2 || array[0] != key {
				t.Fatalf("invalid preview envelope: %#v", result)
			}
			groups, ok := array[1].([]interface{})
			if !ok || groups == nil || len(groups) != 0 {
				t.Fatalf("preview must be empty array, not invented rewards: %#v", array[1])
			}
		})
	}
}

func TestGetJXHDAwardRejectsInvalidRequests(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	ctx := &rtmp.RPCContext{CharacterID: "1"}
	for _, args := range [][]interface{}{nil, {"Unknown"}, {float64(1)}, {"Wawagame", "extra"}} {
		if _, err := h.GetJXHDAward(ctx, args); !errors.Is(err, pkgerrors.ErrInvalidArgs) {
			t.Fatalf("args=%#v error=%v", args, err)
		}
	}
	if _, err := h.GetJXHDAward(nil, []interface{}{"Wawagame"}); !errors.Is(err, pkgerrors.ErrUnauthorized) {
		t.Fatalf("nil context error=%v", err)
	}
}

func TestGetJXHDAwardRegisteredWithDispatcher(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	dispatcher := rtmp.NewRPCDispatcher(zap.NewNop())
	h.RegisterHandlers(dispatcher)
	conn := rtmp.NewConnection(1, nil, dispatcher, zap.NewNop())
	conn.SetCharacter("1")
	result, err := dispatcher.Dispatch(conn, "getJXHDAward", []interface{}{"Wawagame"})
	if err != nil || result == nil {
		t.Fatalf("dispatch result=%#v error=%v", result, err)
	}
}
