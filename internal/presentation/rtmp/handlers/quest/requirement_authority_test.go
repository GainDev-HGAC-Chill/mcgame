package quest

import (
	"context"
	"testing"

	appitem "mcgame-server/internal/application/item"
	appquest "mcgame-server/internal/application/quest"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func TestTakeBuildQuestNeverAcceptsClientProgressOrEmptyDefinitions(t *testing.T) {
	for _, request := range [][]interface{}{
		{float64(8000), float64(430), []interface{}{}},
		{float64(8000), float64(430), []interface{}{map[string]interface{}{"type": float64(2), "target": float64(335), "current": float64(999), "required": float64(0)}}},
		{map[string]interface{}{"id": float64(8000), "nid": float64(430)}},
	} {
		logger := zap.NewNop()
		manager := gamedata.NewManager(nil, logger)
		loadLifecycleQuestTestData(t, manager, 8000, 335, "Required item")
		repo := newLifecycleTestQuestRepo()
		items := appitem.NewService(newLifecycleTestItemRepo(), logger)
		items.SetGameDataManager(manager)
		svc := appquest.NewService(repo, logger)
		svc.SetGameDataManager(manager)
		svc.SetCharacterRepository(&lifecycleTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}})
		svc.SetItemService(items)
		h := NewHandler(svc, logger)
		sender := &finishCallbackTransport{}
		conn := rtmp.NewConnection(1, nil, nil, logger)
		conn.SetTransport(sender)
		ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1", Connection: conn}
		if _, err := h.TakeBuildQuest(ctx, request); err != nil {
			t.Fatal(err)
		}
		qp, err := svc.GetActiveQuest(ctx.Context, 1, 8000)
		if err != nil {
			t.Fatal(err)
		}
		if len(qp.Objectives) != 1 || qp.Objectives[0].Target != 335 || qp.Objectives[0].Required != 1 || qp.Objectives[0].Current != 0 || qp.CanComplete() {
			t.Fatalf("trusted request objectives: %+v", qp)
		}
		if _, _, err := svc.CompleteQuest(ctx.Context, 1, 8000); err == nil {
			t.Fatal("empty inventory could collect quest reward")
		}
	}
}
