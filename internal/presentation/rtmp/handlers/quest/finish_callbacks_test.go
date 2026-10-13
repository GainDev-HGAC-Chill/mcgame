package quest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	appitem "mcgame-server/internal/application/item"
	appquest "mcgame-server/internal/application/quest"
	domainchar "mcgame-server/internal/domain/character"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type finishCallbackTransport struct {
	calls []petRewardCallbackCall
}

func (s *finishCallbackTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	s.calls = append(s.calls, petRewardCallbackCall{method: method, args: args})
	return nil
}

func (s *finishCallbackTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return s.WriteCommand(ctx, method, args)
}

func (s *finishCallbackTransport) WriteResult(context.Context, int64, interface{}) error { return nil }
func (s *finishCallbackTransport) WriteError(context.Context, int64, error) error        { return nil }
func (s *finishCallbackTransport) Kind() string                                          { return "test" }
func (s *finishCallbackTransport) Close() error                                          { return nil }

func TestFinishQuestCallbackContractsAndConsumedInventory(t *testing.T) {
	for _, stack := range []int{1, 2} {
		t.Run(map[int]string{1: "full consumption", 2: "partial consumption"}[stack], func(t *testing.T) {
			logger := zap.NewNop()
			manager := gamedata.NewManager(nil, logger)
			loadLifecycleQuestTestData(t, manager, 160, 335, "Quest item")
			if err := manager.GetCache().LoadTable(models.TableNpc, []json.RawMessage{mustLifecycleJSON(t, models.NpcTemplate{ID: 430, PosMapID: 1})}); err != nil {
				t.Fatal(err)
			}
			charRepo := &lifecycleTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30, MapID: 1}}
			itemRepo := newLifecycleTestItemRepo()
			itemRepo.nextID = 7001
			questRepo := newLifecycleTestQuestRepo()
			items := appitem.NewService(itemRepo, logger)
			quests := appquest.NewService(questRepo, logger)
			items.SetGameDataManager(manager)
			items.SetCharacterRepository(charRepo)
			items.SetQuestService(quests)
			quests.SetGameDataManager(manager)
			quests.SetCharacterRepository(charRepo)
			quests.SetItemService(items)
			h := NewHandler(quests, logger)
			h.SetItemService(items)
			h.SetGameDataManager(manager)
			sender := &finishCallbackTransport{}
			conn := rtmp.NewConnection(1, nil, nil, logger)
			conn.SetTransport(sender)
			ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1", Connection: conn}
			if _, err := h.TakeQuest(ctx, []interface{}{float64(160), float64(430)}); err != nil {
				t.Fatal(err)
			}
			item := itemRepo.items[7001]
			if item == nil {
				t.Fatal("quest start item missing")
			}
			item.StackCount = stack
			sid := item.CalculateSID()
			sender.calls = nil
			result, err := h.FinishQuest(ctx, []interface{}{map[string]interface{}{"i": float64(160)}})
			if err != nil || result.(map[string]interface{})["flag"] != true {
				t.Fatalf("finish result=%#v error=%v", result, err)
			}
			progress, err := questRepo.FindByCharacterAndQuest(ctx.Context, 1, 160)
			if err != nil || progress.Status != domainquest.QuestStatusCompleted {
				t.Fatalf("quest did not persist completion: %#v %v", progress, err)
			}
			deletions, updates := 0, 0
			for _, call := range sender.calls {
				switch call.method {
				case "onUpdateAchieve", "onUpdateSafeBattleMid", "lvUp":
					t.Fatalf("invalid placeholder callback: %s %#v", call.method, call.args)
				case "onDelCharactorSlot":
					deletions++
					if len(call.args) != 2 || call.args[0] != int64(7001) || call.args[1] != sid {
						t.Fatalf("deletion must carry item ID and SID: %#v", call.args)
					}
				case "onAddCharactorSlot":
					updates++
					if len(call.args) != 1 {
						t.Fatalf("slot update arguments: %#v", call.args)
					}
				}
			}
			if stack == 1 {
				if deletions != 1 || itemRepo.items[7001] != nil {
					t.Fatal("fully consumed item was not removed exactly once")
				}
			} else if deletions != 0 || updates != 1 || itemRepo.items[7001].StackCount != 1 {
				t.Fatal("partial consumption must update remaining stack without deleting")
			}
			sender.calls = nil
			repeat, err := h.FinishQuest(ctx, []interface{}{float64(160)})
			if err != nil || repeat.(map[string]interface{})["flag"] != false || len(sender.calls) != 0 {
				t.Fatalf("repeated claim should fail without callbacks: %#v %v %#v", repeat, err, sender.calls)
			}
			if info, ok := repeat.(map[string]interface{})["info"].(string); !ok || info == "" {
				t.Fatal("rejected finish omitted Flash responder info")
			}
		})
	}
}

func TestFinishQuestRewardRefreshesAnotherQuestAfterSlotUpdate(t *testing.T) {
	logger := zap.NewNop()
	manager := gamedata.NewManager(nil, logger)
	loadLifecycleQuestTestData(t, manager, 160, 335, "Reward quest item")
	for table, record := range map[string]interface{}{
		models.TableNpc:          models.NpcTemplate{ID: 430, PosMapID: 1},
		models.TableQuest:        models.QuestTemplate{ID: 8000, Type: 1, FinishNPC: 430},
		models.TableQuestRequire: models.QuestRequireTemplate{ID: 8000, Qid: 8000, Kind: 1, Type: 29, ItemID: 335, Num: 1, Q: -1},
		models.TableQuestAward:   models.QuestAwardTemplate{ID: 8000, Qid: 160, ItemID: 335, Type: 29, Num: 1},
	} {
		if err := manager.GetCache().LoadTable(table, []json.RawMessage{mustLifecycleJSON(t, record)}); err != nil {
			t.Fatal(err)
		}
	}
	chars := &lifecycleTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30, MapID: 1}}
	repo := newLifecycleTestQuestRepo()
	items := appitem.NewService(newLifecycleTestItemRepo(), logger)
	quests := appquest.NewService(repo, logger)
	items.SetGameDataManager(manager)
	items.SetCharacterRepository(chars)
	items.SetQuestService(quests)
	quests.SetGameDataManager(manager)
	quests.SetCharacterRepository(chars)
	quests.SetItemService(items)
	h := NewHandler(quests, logger)
	h.SetItemService(items)
	h.SetGameDataManager(manager)
	h.SetQuestNotifyHelper(NewQuestNotifyHelper(quests, manager, logger))
	sender := &finishCallbackTransport{}
	conn := rtmp.NewConnection(1, nil, nil, logger)
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1", Connection: conn}
	if _, err := h.TakeQuest(ctx, []interface{}{float64(160), float64(430)}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx.Context, domainquest.NewQuestProgress(1, 8000, nil)); err != nil {
		t.Fatal(err)
	}
	sender.calls = nil
	done := make(chan error, 1)
	go func() {
		result, err := h.FinishQuest(ctx, []interface{}{float64(160)})
		if err == nil && result.(map[string]interface{})["flag"] != true {
			err = fmt.Errorf("finish failed: %v", result)
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("finish RPC deadlocked")
	}
	slotIndex, progressIndex := -1, -1
	for i, call := range sender.calls {
		if call.method == "onAddCharactorSlot" {
			slotIndex = i
		}
		if call.method == "onAddChaQuest" {
			payload := call.args[0].(map[string]interface{})
			if payload["qid"] == "8000" {
				progressIndex = i
				if payload["d"] == nil {
					t.Fatal("missing established quest envelope")
				}
			}
		}
	}
	if slotIndex < 0 || progressIndex <= slotIndex {
		t.Fatalf("progress not refreshed after reward slot: %#v", sender.calls)
	}
}
