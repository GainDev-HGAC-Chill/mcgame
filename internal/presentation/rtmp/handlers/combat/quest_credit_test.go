package combat

import (
	"context"
	"encoding/json"
	"testing"

	appcombat "mcgame-server/internal/application/combat"
	appquest "mcgame-server/internal/application/quest"
	combatdomain "mcgame-server/internal/domain/combat"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	questhandler "mcgame-server/internal/presentation/rtmp/handlers/quest"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type questCreditBattleRepo struct {
	combatdomain.BattleRepository
	battle *combatdomain.Battle
}

func (r *questCreditBattleRepo) Remove(string) error {
	return nil
}

func (r *questCreditBattleRepo) GetByID(string) (*combatdomain.Battle, error) {
	if r.battle != nil {
		return r.battle, nil
	}
	return nil, pkgerrors.ErrNotFound
}

type questCreditBattleLogRepo struct {
	combatdomain.BattleLogRepository
}

func (r *questCreditBattleLogRepo) Save(context.Context, *combatdomain.Battle) error {
	return nil
}

func newQuestCreditCombatService(battle *combatdomain.Battle) *appcombat.Service {
	service := appcombat.NewService(&questCreditBattleRepo{}, &questCreditBattleLogRepo{}, nil, nil, nil, nil, zap.NewNop())
	service.EndBattle(context.Background(), battle, combatdomain.SidePlayer)
	return service
}

func newQuestCreditQuestService(t *testing.T, repo domainquest.Repository, required int) *appquest.Service {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	for table, record := range map[string]interface{}{
		models.TableQuest:        models.QuestTemplate{ID: 50, Type: 1},
		models.TableCreature:     models.CreatureTemplate{ID: 157, Name: "Kê Vương"},
		models.TableQuestRequire: models.QuestRequireTemplate{ID: 1, Qid: 50, Kind: 2, Type: 12, ItemID: 157, Num: float64(required)},
	} {
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := manager.GetCache().LoadTable(table, []json.RawMessage{data}); err != nil {
			t.Fatal(err)
		}
	}
	svc := appquest.NewService(repo, zap.NewNop())
	svc.SetGameDataManager(manager)
	return svc
}

func TestEndBattle_RejectsForeignBattleBeforeFinalization(t *testing.T) {
	battle := combatdomain.NewBattle(combatdomain.BattleTypePVE, 3)
	player := combatdomain.NewParticipant("1", "player", combatdomain.SidePlayer, false)
	player.EntityID = 1
	player.EntityType = combatdomain.ParticipantTypeCharacter
	battle.AddParticipant(player)
	battle.Start()
	handler := &Handler{
		combatService: appcombat.NewService(&questCreditBattleRepo{battle: battle}, nil, nil, nil, nil, nil, zap.NewNop()),
		logger:        zap.NewNop(),
	}
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "99"}
	response, err := handler.EndBattle(ctx, []interface{}{battle.ID})
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || !battle.IsActive() || battle.Result != nil {
		t.Fatalf("foreign cleanup changed battle: response=%#v", response)
	}
}

func TestCollectQuestKillUpdates_RejectsUnclaimedBattle(t *testing.T) {
	repo := &combatHandlerQuestRepo{progress: map[int64]*domainquest.QuestProgress{
		1: {ID: 1, CharacterID: 1, QuestID: 50, Status: domainquest.QuestStatusActive,
			Objectives: []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 157, Required: 3}}},
	}}
	handler := &Handler{
		combatService: appcombat.NewService(nil, nil, nil, nil, nil, nil, zap.NewNop()),
		questService:  appquest.NewService(repo, zap.NewNop()),
		logger:        zap.NewNop(),
	}
	battle := combatdomain.NewBattle(combatdomain.BattleTypePVE, 3)
	enemy := combatdomain.NewParticipant("npc_5", "enemy", combatdomain.SideEnemy, true)
	enemy.EntityType = combatdomain.ParticipantTypeCreature
	enemy.EntityID = 157
	battle.AddParticipant(enemy)
	battle.End(&combatdomain.BattleResult{WinnerSide: combatdomain.SidePlayer})
	updated, kills, err := handler.collectQuestKillUpdates(context.Background(), 1, battle)
	if len(updated) != 0 || len(kills) != 0 || err != nil || repo.progress[1].Objectives[0].Current != 0 {
		t.Fatal("uncached battle granted quest credit")
	}
}

func TestBattleQuestProgressEnvelope_UsesSharedBuilder(t *testing.T) {
	progress := &domainquest.QuestProgress{QuestID: 50}
	dto := map[string]interface{}{"state": 1, "pos": 2, "clsData": map[string]interface{}{}}
	payload := questhandler.BuildAddQuestPayload(progress, dto, 3)
	if payload["qid"] != "50" || payload["st"] != 1 || payload["qn"] != 3 {
		t.Fatalf("quest envelope = %#v", payload)
	}
	data, ok := payload["d"].(map[string]interface{})
	if !ok || data["pos"] != 2 || data["clsData"] == nil {
		t.Fatalf("missing client quest data: %#v", payload)
	}
}
