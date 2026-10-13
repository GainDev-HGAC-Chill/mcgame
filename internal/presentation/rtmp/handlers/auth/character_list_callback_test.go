package auth

import (
	"context"
	"reflect"
	"testing"

	appchar "mcgame-server/internal/application/character"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type characterListTransport struct {
	methods []string
	args    [][]interface{}
}

func (s *characterListTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	s.methods = append(s.methods, method)
	s.args = append(s.args, args)
	return nil
}

func (s *characterListTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return s.WriteCommand(ctx, method, args)
}

func (s *characterListTransport) WriteResult(context.Context, int64, interface{}) error {
	return nil
}

func (s *characterListTransport) WriteError(context.Context, int64, error) error {
	return nil
}

func (s *characterListTransport) Kind() string { return "test" }

func (s *characterListTransport) Close() error { return nil }

type characterListLineProvider struct {
	lines []map[string]interface{}
}

func (p *characterListLineProvider) GetLineList(context.Context) ([]map[string]interface{}, error) {
	return p.lines, nil
}

func characterListContext(app string) (*rtmp.RPCContext, *characterListTransport) {
	sender := &characterListTransport{}
	conn := rtmp.NewConnection(1, nil, nil, zap.NewNop())
	conn.SetAppName(app)
	conn.SetTransport(sender)
	return &rtmp.RPCContext{Context: context.Background(), ConnID: 1, Connection: conn}, sender
}

func TestGetLineInfoReturnsLinesWithoutCharacterCallbacks(t *testing.T) {
	for _, app := range []string{"tcn/", "master/test/", "scene/"} {
		t.Run(app, func(t *testing.T) {
			charService := appchar.NewService(&authTestCharacterRepo{}, zap.NewNop())
			h := NewHandler(nil, charService, nil, nil, nil, nil, nil, zap.NewNop())
			lines := []map[string]interface{}{{"id": 1, "name": "Test line"}}
			h.SetLineListProvider(&characterListLineProvider{lines: lines})
			ctx, sender := characterListContext(app)
			result, err := h.GetLineInfo(ctx, nil)
			if err != nil || !reflect.DeepEqual(result, lines) {
				t.Fatalf("result=%#v error=%v", result, err)
			}
			if len(sender.methods) != 0 {
				t.Fatalf("getLineInfo sent callbacks: %v", sender.methods)
			}
		})
	}
}

func TestOnIclCallbackOnlySentToSceneConnection(t *testing.T) {
	for _, app := range []string{"tcn", "tcn/", "master/test", "master/test/", "scene/", "line1/"} {
		t.Run(app, func(t *testing.T) {
			h := NewHandler(nil, nil, nil, nil, nil, nil, nil, zap.NewNop())
			ctx, sender := characterListContext(app)
			list := []map[string]interface{}{{"id": int64(1), "name": "Gain"}}
			h.sendOnIclCallback(ctx, list)
			if ctx.Connection.IsGlobalConnection() {
				if len(sender.methods) != 0 {
					t.Fatalf("global connection received callbacks: %v", sender.methods)
				}
				return
			}
			if len(sender.methods) != 1 || sender.methods[0] != "onIcl" {
				t.Fatalf("scene callbacks: %v", sender.methods)
			}
			if len(sender.args[0]) != 1 || !reflect.DeepEqual(sender.args[0][0], list) {
				t.Fatalf("scene character list changed: %#v", sender.args)
			}
		})
	}
}
