package quest

import (
	"context"
	"encoding/json"
	"testing"

	appquest "mcgame-server/internal/application/quest"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func TestQuestCallbacksPreserveLeadingSortFieldAndReplaceExistingQuest(t *testing.T) {
	logger := zap.NewNop()
	manager := gamedata.NewManager(nil, logger)
	templates := []json.RawMessage{
		mustLifecycleJSON(t, models.QuestTemplate{ID: 3165, Name: "Khiêu chiến chiêu tài bảo", Type: 10, Color: 0}),
		mustLifecycleJSON(t, models.QuestTemplate{ID: 3166, Name: "Existing quest", Type: 10, Color: 3}),
	}
	requirements := []json.RawMessage{
		mustLifecycleJSON(t, models.QuestRequireTemplate{ID: 1, Qid: 3165, Kind: 2, Type: 12, ItemID: 126, Num: 1}),
		mustLifecycleJSON(t, models.QuestRequireTemplate{ID: 2, Qid: 3166, Kind: 2, Type: 12, ItemID: 126, Num: 1}),
	}
	for table, rows := range map[string][]json.RawMessage{
		models.TableQuest:        templates,
		models.TableQuestRequire: requirements,
		models.TableCreature:     {mustLifecycleJSON(t, models.CreatureTemplate{ID: 126, Name: "Chieu Tai Mao"})},
	} {
		if err := manager.GetCache().LoadTable(table, rows); err != nil {
			t.Fatal(err)
		}
	}
	repo := newLifecycleTestQuestRepo()
	progress := domainquest.NewQuestProgress(1, 3165, []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 126, Required: 1}})
	other := domainquest.NewQuestProgress(1, 3166, []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 126, Required: 1}})
	for _, qp := range []*domainquest.QuestProgress{progress, other} {
		if err := repo.Save(context.Background(), qp); err != nil {
			t.Fatal(err)
		}
	}
	svc := appquest.NewService(repo, logger)
	svc.SetGameDataManager(manager)
	handler := NewHandler(svc, logger)
	sender := &finishCallbackTransport{}
	conn := rtmp.NewConnection(1, nil, nil, logger)
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1", Connection: conn}
	clientQuests := make(map[interface{}]map[string]interface{})
	consume := func() {
		t.Helper()
		for _, call := range sender.calls {
			if call.method != "onAddChaQuest" || len(call.args) != 1 {
				t.Fatalf("unexpected callback: %#v", call)
			}
			payload := call.args[0].(map[string]interface{})
			body := payload["d"].(map[string]interface{})
			data := body["data"].(map[string]interface{})
			if _, ok := data["color"].(int); !ok {
				t.Fatalf("Flex sort1 undefined/non-numeric while sort2=%#v: %#v", body["state"], data)
			}
			if _, ok := body["state"].(int); !ok || payload["qn"] != 2 {
				t.Fatalf("invalid state/count: %#v", payload)
			}
			clientQuests[data["id"]] = body
		}
		sender.calls = nil
	}
	handler.sendAddQuestCallback(ctx, 1, progress)
	handler.sendAddQuestCallback(ctx, 1, other)
	consume()
	progress.Objectives[0].Current = 1
	if err := repo.Update(ctx.Context, progress); err != nil {
		t.Fatal(err)
	}
	helper := NewQuestNotifyHelper(svc, manager, logger)
	helper.RefreshQuestProgressSilently(ctx, 1)
	if len(sender.calls) != 0 {
		t.Fatalf("silent quest refresh emitted callbacks: %#v", sender.calls)
	}
	for range 2 {
		helper.PushQuestProgress(ctx, 1)
		if len(sender.calls) != 2 {
			t.Fatalf("expected full active-quest refresh, got %#v", sender.calls)
		}
		consume()
	}
	if len(clientQuests) != 2 || clientQuests[int64(3165)]["state"] != 104 {
		t.Fatalf("duplicate add must replace, not append, and retain new state: %#v", clientQuests)
	}
	result, err := handler.InitQuestManager(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	quests := result.(map[string]interface{})["q"].(map[string]interface{})
	for _, value := range quests {
		body := value.(map[string]interface{})
		if _, ok := body["data"].(map[string]interface{})["color"].(int); !ok {
			t.Fatalf("initial snapshot must also preserve sort1: %#v", body)
		}
	}
}
