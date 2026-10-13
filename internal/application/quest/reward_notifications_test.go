package quest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appitem "mcgame-server/internal/application/item"
	domainchar "mcgame-server/internal/domain/character"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestCompleteQuestItemRewardWithBidirectionalServicesDoesNotDeadlock(t *testing.T) {
	svc := specialQuestService(t, 8000, "Catalog kill", 1)
	loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{ID: 1, Qid: 8000, ItemID: 9001, Kind: 2, Type: 12, Num: 1})
	for table, record := range map[string]interface{}{
		models.TableQuest:      models.QuestTemplate{ID: 8001, Type: 1},
		models.TableQuestAward: models.QuestAwardTemplate{ID: 1, Qid: 8000, ItemID: 30, Type: 29, Num: 1},
	} {
		if err := svc.gameDataManager.GetCache().LoadTable(table, []json.RawMessage{mustQuestJSON(t, record)}); err != nil {
			t.Fatal(err)
		}
	}
	loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{ID: 2, Qid: 8001, ItemID: 30, Kind: 1, Type: 29, Num: 1, Q: -1})
	items := appitem.NewService(newQuestTestItemRepo(), zap.NewNop())
	items.SetGameDataManager(svc.gameDataManager)
	items.SetQuestService(svc)
	svc.SetItemService(items)
	svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, Level: 90}})
	authoritySave(t, svc, domainquest.NewQuestProgress(2, 8000, []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 1, Current: 1}}))
	authoritySave(t, svc, domainquest.NewQuestProgress(2, 8001, nil))
	done := make(chan error, 1)
	go func() { _, _, err := svc.CompleteQuest(context.Background(), 2, 8000); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("completion deadlocked on reward item notification")
	}
	stored, err := svc.questRepo.FindByCharacterAndQuest(context.Background(), 2, 8001)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Objectives) != 1 || stored.Objectives[0].Current != 1 {
		t.Fatalf("reward did not advance other quest: %+v", stored)
	}
	if _, _, err := svc.CompleteQuest(context.Background(), 2, 8000); err == nil {
		t.Fatal("completed quest rewarded twice")
	}
}

func TestLevelNotificationUsesReadOnlyAuthoritativeRequirements(t *testing.T) {
	svc := specialQuestService(t, 2590, "Cấp Độ Lên Đến 10", 10)
	svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, Level: 1}})
	authoritySave(t, svc, domainquest.NewQuestProgress(2, 2590, []domainquest.Objective{{Type: domainquest.ObjectiveReachLevel, Target: 10, Required: 1}}))
	repo := &failingCreditRepo{questTestQuestRepo: svc.questRepo.(*questTestQuestRepo)}
	svc.questRepo = repo
	if err := svc.OnLevelUp(context.Background(), 2, 1, 999); err != nil {
		t.Fatal(err)
	}
	if repo.writes != 0 {
		t.Fatal("level notification persisted a stale quest snapshot")
	}
	qp, err := svc.GetQuestProgress(context.Background(), 2, 2590)
	if err != nil {
		t.Fatal(err)
	}
	if qp.CanComplete() {
		t.Fatal("untrusted level notification bypassed actual character level")
	}
}
