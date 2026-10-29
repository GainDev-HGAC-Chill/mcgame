package npc

import (
	"context"
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"
)

type weddingBookProviderTest struct {
	called bool
}

func (p *weddingBookProviderTest) SendWeddingBookList(context.Context, *rtmp.Connection) error {
	p.called = true
	return nil
}

func TestClickNpcWeddingManagerInitializesBookList(t *testing.T) {
	provider := &weddingBookProviderTest{}
	handler := &Handler{weddingBookProvider: provider}
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "3"}
	response, err := handler.ClickNpc(ctx, []interface{}{float64(1281)})
	if err != nil {
		t.Fatal(err)
	}
	if response != nil || !provider.called {
		t.Fatalf("response = %#v, called = %t", response, provider.called)
	}
}
