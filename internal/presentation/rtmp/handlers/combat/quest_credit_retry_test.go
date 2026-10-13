package combat

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	appcombat "mcgame-server/internal/application/combat"
	combatdomain "mcgame-server/internal/domain/combat"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type retryCreditQuestRepo struct {
	*combatHandlerQuestRepo
	updateCount int
	failAt      int
	failErr     error
	entered     chan struct{}
	unblock     chan struct{}
}

func (r *retryCreditQuestRepo) Update(ctx context.Context, progress *domainquest.QuestProgress) error {
	r.updateCount++
	if r.entered != nil {
		close(r.entered)
		<-r.unblock
		r.entered = nil
	}
	if r.updateCount == r.failAt {
		return r.failErr
	}
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	var persisted domainquest.QuestProgress
	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}
	r.progress[progress.ID] = &persisted
	return nil
}

func (r *retryCreditQuestRepo) FindActiveByCharacter(ctx context.Context, characterID int64) ([]*domainquest.QuestProgress, error) {
	progress, err := r.combatHandlerQuestRepo.FindActiveByCharacter(ctx, characterID)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(progress)
	if err != nil {
		return nil, err
	}
	var detached []*domainquest.QuestProgress
	if err := json.Unmarshal(data, &detached); err != nil {
		return nil, err
	}
	return detached, nil
}

func retryCreditFixture(t *testing.T) (*Handler, *combatdomain.Battle, *retryCreditQuestRepo) {
	repo := &retryCreditQuestRepo{combatHandlerQuestRepo: &combatHandlerQuestRepo{
		progress: map[int64]*domainquest.QuestProgress{
			1: {ID: 1, CharacterID: 1, QuestID: 50, Status: domainquest.QuestStatusActive,
				Objectives: []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 157, Required: 10}}},
		},
	}}
	battle := combatdomain.NewBattle(combatdomain.BattleTypePVE, 3)
	player := combatdomain.NewParticipant("1", "player", combatdomain.SidePlayer, false)
	player.EntityID = 1
	player.EntityType = combatdomain.ParticipantTypeCharacter
	player.IsAlive = false
	battle.AddParticipant(player)
	for _, id := range []string{"creature_1", "creature_2", "survivor", "not_dead"} {
		enemy := combatdomain.NewParticipant(id, "enemy", combatdomain.SideEnemy, true)
		enemy.EntityType = combatdomain.ParticipantTypeCreature
		enemy.EntityID = 157
		enemy.IsAlive = id == "survivor"
		if id == "not_dead" {
			enemy.CurrentHP = 10
		}
		battle.AddParticipant(enemy)
	}
	handler := &Handler{
		combatService: newQuestCreditCombatService(battle),
		questService:  newQuestCreditQuestService(t, repo, 10),
		logger:        zap.NewNop(),
	}
	return handler, battle, repo
}

func TestQuestKillCredit_FailedSuffixRetriesWithoutDuplicatingPrefix(t *testing.T) {
	handler, battle, repo := retryCreditFixture(t)
	repo.failAt = 2
	repo.failErr = errors.New("transient quest write failure")
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1"}
	if _, err := handler.EndBattle(ctx, []interface{}{battle.ID}); !errors.Is(err, repo.failErr) {
		t.Fatalf("cleanup failure = %v, want transient error", err)
	}
	if _, err := handler.combatService.GetBattleForCleanup(battle.ID); err != nil {
		t.Fatalf("failed cleanup removed battle: %v", err)
	}
	if got := repo.progress[1].Objectives[0].Current; got != 1 {
		t.Fatalf("successful prefix = %d, want 1", got)
	}
	_, kills, err := handler.collectQuestKillUpdates(context.Background(), 1, battle)
	if err != nil {
		t.Fatal(err)
	}
	if got := repo.progress[1].Objectives[0].Current; got != 2 {
		t.Fatalf("retried objective count = %d, want 2 unique dead creatures", got)
	}
	if kills["157"] != 2 {
		t.Fatalf("dead creature callback count = %#v, want 2", kills)
	}
	if _, repeated, err := handler.collectQuestKillUpdates(context.Background(), 1, battle); err != nil || len(repeated) != 0 {
		t.Fatalf("completed replay = %#v, %v", repeated, err)
	}
	handler.combatService.AckBattleCleanup(battle.ID, 1)
	if _, err := handler.combatService.GetBattleForCleanup(battle.ID); err == nil {
		t.Fatal("successful cleanup retained battle")
	}
}

func TestEndBattle_InflightCreditRejectsConcurrentCleanupWithoutAck(t *testing.T) {
	handler, battle, repo := retryCreditFixture(t)
	entered := make(chan struct{})
	unblock := make(chan struct{})
	repo.entered = entered
	repo.unblock = unblock
	finished := make(chan error, 1)
	go func() {
		_, _, err := handler.collectQuestKillUpdates(context.Background(), 1, battle)
		finished <- err
	}()
	<-entered
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "1"}
	_, err := handler.EndBattle(ctx, []interface{}{battle.ID})
	if !errors.Is(err, appcombat.ErrQuestKillCreditInflight) {
		close(unblock)
		<-finished
		t.Fatalf("concurrent cleanup error = %v, want inflight", err)
	}
	handler.combatService.AckBattleCleanup(battle.ID, 1)
	_, lookupErr := handler.combatService.GetBattleForCleanup(battle.ID)
	close(unblock)
	creditErr := <-finished
	if lookupErr != nil || creditErr != nil {
		t.Fatalf("inflight lifecycle errors: lookup=%v credit=%v", lookupErr, creditErr)
	}
	if got := repo.progress[1].Objectives[0].Current; got != 2 {
		t.Fatalf("credited creatures = %d, want 2", got)
	}
	handler.combatService.AckBattleCleanup(battle.ID, 1)
	if _, err := handler.combatService.GetBattleForCleanup(battle.ID); err == nil {
		t.Fatal("completed cleanup retained battle")
	}
}
