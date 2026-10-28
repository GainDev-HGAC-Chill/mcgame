// Guild building NPC callbacks follow the shipped Flash callback traits.
package npc

import (
	"context"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type guildBuildingCallback struct {
	method string
	args   []interface{}
}

func TestFindGuildBuildingAtExtensionReturnsPersistedPendingState(t *testing.T) {
	want := &domainguild.BuildingState{ID: 9, ExtendID: 10, PendingTemplateID: 2, Mode: 2}
	buildings := []*domainguild.BuildingState{nil, {ID: 8, ExtendID: 9, TemplateID: 5}, want}

	got := findGuildBuildingAtExtension(buildings, 10)
	if got != want {
		t.Fatalf("findGuildBuildingAtExtension() = %#v, want pending building %#v", got, want)
	}
	if templateID := buildingClientTemplateID(got); templateID != 2 {
		t.Fatalf("buildingClientTemplateID() = %d, want 2", templateID)
	}
}

type guildBuildingTransport struct {
	calls []guildBuildingCallback
}

func (t *guildBuildingTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	t.calls = append(t.calls, guildBuildingCallback{method: method, args: args})
	return nil
}

func (t *guildBuildingTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return t.WriteCommand(ctx, method, args)
}

func (t *guildBuildingTransport) WriteResult(context.Context, int64, interface{}) error {
	return nil
}

func (t *guildBuildingTransport) WriteError(context.Context, int64, error) error {
	return nil
}

func (t *guildBuildingTransport) Kind() string {
	return "test"
}

func (t *guildBuildingTransport) Close() error {
	return nil
}

func TestNpcScriptStartConstructionUsesShippedCallback(t *testing.T) {
	conn := rtmp.NewConnection(1, nil, nil, zap.NewNop())
	conn.SetCharacter("3")
	conn.SetLastGuildBuildContext(guildBuildConstructID, guildBuildConstructType)
	transport := &guildBuildingTransport{}
	conn.SetTransport(transport)
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "3", Connection: conn}

	result, err := h.NpcScript(ctx, []interface{}{"startConstruction"})
	if err != nil {
		t.Fatalf("NpcScript() error = %v", err)
	}
	if result != nil {
		t.Fatalf("NpcScript() result = %#v, want nil for fire-and-forget client call", result)
	}
	if len(transport.calls) != 1 {
		t.Fatalf("callback count = %d, want 1", len(transport.calls))
	}
	call := transport.calls[0]
	if call.method != "onShowNewBuildManager" {
		t.Fatalf("callback method = %q, want onShowNewBuildManager", call.method)
	}
	if len(call.args) != 1 || call.args[0] != guildBuildConstructID {
		t.Fatalf("callback args = %#v, want [%d]", call.args, guildBuildConstructID)
	}
}

var _ rtmp.Transport = (*guildBuildingTransport)(nil)
