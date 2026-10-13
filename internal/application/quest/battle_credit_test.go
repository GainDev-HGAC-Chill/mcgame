package quest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	appitem "mcgame-server/internal/application/item"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/social"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type failingCreditRepo struct {
	*questTestQuestRepo
	writes int
	failAt int
}

func (r *failingCreditRepo) Update(ctx context.Context, qp *domainquest.QuestProgress) error {
	r.writes++
	if r.writes == r.failAt {
		return errors.New("injected quest write failure")
	}
	return r.questTestQuestRepo.Update(ctx, qp)
}

func TestBattleQuestCreditRetryDoesNotReplaySuccessfulPrefix(t *testing.T) {
	svc := specialQuestService(t, 8000, "Catalog kill", 1)
	if err := svc.gameDataManager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, models.QuestTemplate{ID: 8001, Type: 1})}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{8000, 8001} {
		loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{Qid: float64(id), ItemID: 9001, Kind: 2, Type: 12, Num: 2})
		authoritySave(t, svc, domainquest.NewQuestProgress(2, id, nil))
	}
	repo := &failingCreditRepo{questTestQuestRepo: svc.questRepo.(*questTestQuestRepo), failAt: 2}
	svc.questRepo = repo
	if _, err := svc.OnMonsterKilledWithCredit(context.Background(), 2, 9001, "battle:enemy1"); err == nil {
		t.Fatal("write failure was swallowed")
	}
	total := 0
	for _, id := range []int{8000, 8001} {
		qp, _ := repo.FindByCharacterAndQuest(context.Background(), 2, id)
		if len(qp.Objectives) > 0 {
			total += qp.Objectives[0].Current
		}
	}
	if total != 1 {
		t.Fatalf("successful prefix=%d want1", total)
	}
	if _, err := svc.OnMonsterKilledWithCredit(context.Background(), 2, 9001, "battle:enemy1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.OnMonsterKilledWithCredit(context.Background(), 2, 9001, "battle:enemy1"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{8000, 8001} {
		qp, _ := repo.FindByCharacterAndQuest(context.Background(), 2, id)
		if qp.Objectives[0].Current != 1 || len(qp.Objectives[0].KillCreditEvents) != 1 {
			t.Fatalf("replayed event: %+v", qp.Objectives)
		}
	}
	if _, err := svc.OnMonsterKilledWithCredit(context.Background(), 2, 9001, "battle:enemy2"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{8000, 8001} {
		qp, _ := repo.FindByCharacterAndQuest(context.Background(), 2, id)
		if qp.Objectives[0].Current != 2 || len(qp.Objectives[0].KillCreditEvents) != 2 {
			t.Fatalf("missing distinct kill: %+v", qp.Objectives)
		}
	}
}

func TestBattleQuestCreditDuplicateRowsRequireDistinctKills(t *testing.T) {
	svc := specialQuestService(t, 8000, "Catalog kill", 1)
	for _, id := range []int64{1, 2} {
		loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{ID: id, Qid: 8000, ItemID: 9001, Kind: 2, Type: 12, Num: 1})
	}
	authoritySave(t, svc, domainquest.NewQuestProgress(2, 8000, nil))
	for range 2 {
		if _, err := svc.OnMonsterKilledWithCredit(context.Background(), 2, 9001, "battle:enemy1"); err != nil {
			t.Fatal(err)
		}
	}
	qp, _ := svc.GetQuestProgress(context.Background(), 2, 8000)
	if qp.Objectives[0].Current != 1 || qp.Objectives[1].Current != 0 {
		t.Fatalf("one event satisfied duplicate targets: %+v", qp.Objectives)
	}
	if _, err := svc.OnMonsterKilledWithCredit(context.Background(), 2, 9001, "battle:enemy2"); err != nil {
		t.Fatal(err)
	}
	qp, _ = svc.GetQuestProgress(context.Background(), 2, 8000)
	if !qp.CanComplete() {
		t.Fatal("two distinct events did not complete both rows")
	}
}

func TestQuestReadSyncNeverOverwritesStoredStatus(t *testing.T) {
	svc := specialQuestService(t, 8000, "Catalog collect", 1)
	loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{Qid: 8000, ItemID: 30, Kind: 1, Type: 29, Num: 1})
	svc.SetItemService(appitem.NewService(newQuestTestItemRepo(), zap.NewNop()))
	qp := domainquest.NewQuestProgress(2, 8000, []domainquest.Objective{{Type: domainquest.ObjectiveCollectItem, Target: 30, Required: 1, Current: 1, ItemTableType: 29}})
	authoritySave(t, svc, qp)
	repo := &failingCreditRepo{questTestQuestRepo: svc.questRepo.(*questTestQuestRepo)}
	svc.questRepo = repo
	stale, err := repo.FindByCharacterAndQuest(context.Background(), 2, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if dto := svc.EnrichQuestDTO(context.Background(), stale); dto["state"] != 103 {
		t.Fatal("missing owned item reported complete")
	}
	if repo.writes != 0 {
		t.Fatal("UI sync performed a write")
	}
	completed, err := repo.FindByCharacterAndQuest(context.Background(), 2, 8000)
	if err != nil {
		t.Fatal(err)
	}
	completed.Complete()
	if err := repo.Update(context.Background(), completed); err != nil {
		t.Fatal(err)
	}
	svc.EnrichQuestDTO(context.Background(), stale)
	stored, _ := repo.FindByCharacterAndQuest(context.Background(), 2, 8000)
	if repo.writes != 1 || stored.Status != domainquest.QuestStatusCompleted {
		t.Fatal("stale UI overwrote completed state")
	}
}

func TestReviewedSpecialQuestCannotConsumeForgedObjectives(t *testing.T) {
	svc := specialQuestService(t, 2695, "Kết giao 1 người bạn", 10)
	svc.SetFriendRepository(&questFriendStub{friends: []*social.Relationship{{CharacterID: 2, OtherID: 3, Intimacy: 50}}})
	svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, Level: 90}})
	items := newQuestTestItemRepo()
	items.items[55] = &domainitem.Item{ID: 55, CharacterID: 2, TemplateID: 30, ItemType: domainitem.ItemTypeQuest, StackCount: 1, SlotType: domainitem.SlotTypeQuestBag}
	svc.SetItemService(appitem.NewService(items, zap.NewNop()))
	qp := domainquest.NewQuestProgress(2, 2695, []domainquest.Objective{{Type: domainquest.ObjectiveCollectItem, Target: 30, Required: 1, Current: 1}})
	authoritySave(t, svc, qp)
	if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 104 || len(qp.Objectives) != 0 {
		t.Fatal("reviewed special retained synthetic objectives")
	}
	if _, _, err := svc.CompleteQuest(context.Background(), 2, 2695); err != nil {
		t.Fatal(err)
	}
	if items.items[55] == nil || items.items[55].StackCount != 1 {
		t.Fatal("special quest consumed unrelated item")
	}
}
