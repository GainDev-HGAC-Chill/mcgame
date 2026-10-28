// Open-sourced by BaoLT

package quest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appitem "mcgame-server/internal/application/item"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/gamedata/predef"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type fakeLoopRepo struct {
	states map[int64]map[int]*domainquest.LoopState
}

func newFakeLoopRepo() *fakeLoopRepo {
	return &fakeLoopRepo{states: make(map[int64]map[int]*domainquest.LoopState)}
}

func (r *fakeLoopRepo) GetAll(ctx context.Context, characterID int64) ([]*domainquest.LoopState, error) {
	result := make([]*domainquest.LoopState, 0)
	for _, st := range r.states[characterID] {
		copyState := *st
		result = append(result, &copyState)
	}
	return result, nil
}

func (r *fakeLoopRepo) Upsert(ctx context.Context, state *domainquest.LoopState) (*domainquest.LoopState, error) {
	if r.states[state.CharacterID] == nil {
		r.states[state.CharacterID] = make(map[int]*domainquest.LoopState)
	}
	copyState := *state
	if existing, ok := r.states[state.CharacterID][state.LoopID]; ok {
		copyState.ID = existing.ID
	} else {
		copyState.ID = int64(len(r.states[state.CharacterID]) + 1)
	}
	r.states[state.CharacterID][state.LoopID] = &copyState
	result := copyState
	return &result, nil
}

func (r *fakeLoopRepo) Deactivate(ctx context.Context, characterID int64, loopID int) (bool, error) {
	st, ok := r.states[characterID][loopID]
	if !ok || !st.Active {
		return false, nil
	}
	st.Active = false
	st.ActiveQuestID = 0
	return true, nil
}

type fakeGuildMembership struct {
	inGuild bool
	err     error
}

type fakeGuildLoopCommandRepo struct {
	*fakeLoopRepo
	command domainquest.GuildLoopCompletionCommand
}

func (r *fakeGuildLoopCommandRepo) CompleteGuildLoopAtomic(ctx context.Context, command domainquest.GuildLoopCompletionCommand) (*domainquest.GuildLoopCompletionResult, error) {
	r.command = command
	result := &domainquest.GuildLoopCompletionResult{
		GuildID: 1, Money: command.Money, MoneyBind: command.MoneyBind,
		GuildContrib: command.Contribution, MemberNormal: command.Contribution,
		GuildExperience: command.GuildExp, Ft: command.ExpectedFt + 1,
		Active: command.NextQuest != nil, ConsumedPetIDs: append([]int64(nil), command.PetIDs...),
	}
	for _, consumed := range command.Items {
		result.ItemStates = append(result.ItemStates, domainquest.GuildLoopItemState{ItemID: consumed.ItemID, Remaining: 1})
	}
	if command.NextQuest != nil {
		next := *command.NextQuest
		next.ID = 999
		result.NextQuest = &next
	}
	return result, nil
}

func (g fakeGuildMembership) IsInGuild(ctx context.Context, characterID int64) (bool, error) {
	return g.inGuild, g.err
}

// fakeDatedQuestRepo layers quest_history completion dates on top of the
// existing questTestQuestRepo fake (defined in service_test.go), mirroring
// the optional completionDateRepository capability that only the real
// postgres-backed repo implements in production.
type fakeDatedQuestRepo struct {
	*questTestQuestRepo
	dates map[int]time.Time
}

func newFakeDatedQuestRepo() *fakeDatedQuestRepo {
	return &fakeDatedQuestRepo{questTestQuestRepo: newQuestTestQuestRepo(), dates: make(map[int]time.Time)}
}

func (r *fakeDatedQuestRepo) RecordHistory(ctx context.Context, history *domainquest.QuestHistory) error {
	if err := r.questTestQuestRepo.RecordHistory(ctx, history); err != nil {
		return err
	}
	r.dates[history.QuestID] = history.CompletedAt
	return nil
}

func (r *fakeDatedQuestRepo) GetCompletedQuestIDsWithDate(ctx context.Context, characterID int64) (map[int]time.Time, error) {
	result := make(map[int]time.Time, len(r.dates))
	for id, at := range r.dates {
		result[id] = at
	}
	return result, nil
}

func (r *fakeDatedQuestRepo) GetCompletedQuestIDs(ctx context.Context, characterID int64) ([]int, error) {
	result := make([]int, 0, len(r.dates))
	for id := range r.dates {
		result = append(result, id)
	}
	return result, nil
}

func TestLoopState_CooldownEndsAtMs_AnchorsAtOriginalTakeDate(t *testing.T) {
	state := &domainquest.LoopState{TakeDateMs: 1_000_000}
	got := state.CooldownEndsAtMs(600)
	want := int64(1_000_000 + 600*1000)
	if got != want {
		t.Fatalf("CooldownEndsAtMs(600) = %d, want %d", got, want)
	}
}

func newLoopTestManager(t *testing.T, loops []models.QuestLoopTemplate, quests []models.QuestTemplate) *gamedata.Manager {
	t.Helper()
	logger := zap.NewNop()
	manager := gamedata.NewManager(nil, logger)

	loopRows := make([]json.RawMessage, 0, len(loops))
	for _, l := range loops {
		loopRows = append(loopRows, mustQuestJSON(t, l))
	}
	if len(loopRows) > 0 {
		if err := manager.GetCache().LoadTable(models.TableQuestLoop, loopRows); err != nil {
			t.Fatalf("load quest loop table: %v", err)
		}
	}

	questRows := make([]json.RawMessage, 0, len(quests))
	for _, q := range quests {
		questRows = append(questRows, mustQuestJSON(t, q))
	}
	if len(questRows) > 0 {
		if err := manager.GetCache().LoadTable(models.TableQuest, questRows); err != nil {
			t.Fatalf("load quest table: %v", err)
		}
	}

	return manager
}

func TestCanTakeLoop_Gates(t *testing.T) {
	logger := zap.NewNop()

	loopTpl := models.QuestLoopTemplate{ID: 4, Nid: 277, Num: 10, Refresh: 600, MinLevel: 30, MaxLevel: 0, Team: 0}
	guildLoopTpl := models.QuestLoopTemplate{ID: loopGuildLid, Nid: 1069, Num: 15, Refresh: 600, MinLevel: 30, MaxLevel: 0, Team: 0}
	manager := newLoopTestManager(t, []models.QuestLoopTemplate{loopTpl, guildLoopTpl}, nil)

	t.Run("level_too_low_rejects", func(t *testing.T) {
		charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 10}}
		svc := NewService(newQuestTestQuestRepo(), logger)
		svc.SetGameDataManager(manager)
		svc.SetCharacterRepository(charRepo)
		svc.SetLoopRepository(newFakeLoopRepo())

		ok, reason := svc.CanTakeLoop(context.Background(), 1, 4)
		if ok {
			t.Fatalf("expected level gate to reject, got ok=true reason=%q", reason)
		}
	})

	t.Run("eligible_level_accepts_with_no_state", func(t *testing.T) {
		charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}}
		svc := NewService(newQuestTestQuestRepo(), logger)
		svc.SetGameDataManager(manager)
		svc.SetCharacterRepository(charRepo)
		svc.SetLoopRepository(newFakeLoopRepo())

		ok, reason := svc.CanTakeLoop(context.Background(), 1, 4)
		if !ok {
			t.Fatalf("expected canTake, got false reason=%q", reason)
		}
	})

	t.Run("guild_lid_requires_membership_and_reward_authority", func(t *testing.T) {
		charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}}
		svc := NewService(newQuestTestQuestRepo(), logger)
		svc.SetGameDataManager(manager)
		svc.SetCharacterRepository(charRepo)
		svc.SetLoopRepository(newFakeLoopRepo())
		svc.SetGuildMembership(fakeGuildMembership{inGuild: false})

		if ok, _ := svc.CanTakeLoop(context.Background(), 1, loopGuildLid); ok {
			t.Fatalf("expected guild gate to reject non-member")
		}

		svc.SetGuildMembership(fakeGuildMembership{inGuild: true})
		if ok, reason := svc.CanTakeLoop(context.Background(), 1, loopGuildLid); ok || reason != errGuildLoopRewardsUnavailable.Error() {
			t.Fatalf("expected guild loop to fail closed without reward authority, got ok=%v reason=%q", ok, reason)
		}
	})

	t.Run("active_instance_rejects_retake", func(t *testing.T) {
		charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}}
		loopRepo := newFakeLoopRepo()
		loopRepo.states[1] = map[int]*domainquest.LoopState{
			4: {CharacterID: 1, LoopID: 4, Active: true, TakeDateMs: nowMs()},
		}
		svc := NewService(newQuestTestQuestRepo(), logger)
		svc.SetGameDataManager(manager)
		svc.SetCharacterRepository(charRepo)
		svc.SetLoopRepository(loopRepo)

		if ok, _ := svc.CanTakeLoop(context.Background(), 1, 4); ok {
			t.Fatalf("expected active-instance gate to reject retake")
		}
	})

	t.Run("cooldown_blocks_until_refresh_elapses", func(t *testing.T) {
		charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}}
		loopRepo := newFakeLoopRepo()
		loopRepo.states[1] = map[int]*domainquest.LoopState{
			4: {CharacterID: 1, LoopID: 4, Active: false, TakeDateMs: nowMs()},
		}
		svc := NewService(newQuestTestQuestRepo(), logger)
		svc.SetGameDataManager(manager)
		svc.SetCharacterRepository(charRepo)
		svc.SetLoopRepository(loopRepo)

		if ok, _ := svc.CanTakeLoop(context.Background(), 1, 4); ok {
			t.Fatalf("expected cooldown gate to reject immediate retake")
		}

		loopRepo.states[1][4].TakeDateMs = nowMs() - int64(loopTpl.Refresh)*1000 - 1000
		if ok, reason := svc.CanTakeLoop(context.Background(), 1, 4); !ok {
			t.Fatalf("expected cooldown gate to clear once refresh elapses, got false reason=%q", reason)
		}
	})
}

func TestPickNextChild_FiltersByPoolLevelAndAvoidsImmediateRepeat(t *testing.T) {
	logger := zap.NewNop()

	children := []models.QuestTemplate{
		{ID: 601, Type: 7, SubType: "7-4", MinLevel: 0, MaxLevel: 0, FinishNPC: 277},
		{ID: 602, Type: 7, SubType: "7-4", MinLevel: 0, MaxLevel: 0, FinishNPC: 277},
		{ID: 603, Type: 7, SubType: "7-4", MinLevel: 80, MaxLevel: 0, FinishNPC: 277},
		{ID: 604, Type: 7, SubType: "7-5", MinLevel: 0, MaxLevel: 0, FinishNPC: 280},
	}
	manager := newLoopTestManager(t, nil, children)

	charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}}
	svc := NewService(newQuestTestQuestRepo(), logger)
	svc.SetGameDataManager(manager)
	svc.SetCharacterRepository(charRepo)

	seen := make(map[int]bool)
	for i := 0; i < 40; i++ {
		child := svc.pickNextChild(context.Background(), 1, 4, 601)
		if child == nil {
			t.Fatalf("expected a picked child, got nil")
		}
		if int(child.ID) == 601 {
			t.Fatalf("pickNextChild returned the excluded quest id 601 while an alternative existed")
		}
		if int(child.ID) == 603 {
			t.Fatalf("pickNextChild returned quest 603 which is above the character's level")
		}
		if int(child.ID) == 604 {
			t.Fatalf("pickNextChild returned quest 604 from pool 5, not pool 4")
		}
		seen[int(child.ID)] = true
	}
	if !seen[602] {
		t.Fatalf("expected quest 602 to be reachable as the only eligible non-excluded candidate")
	}
}

func TestPickNextChild_PrefersFightableRoundsAndFallsBackWhenNoneExist(t *testing.T) {
	logger := zap.NewNop()

	children := []models.QuestTemplate{
		{ID: 701, Type: 7, SubType: "7-4", FinishNPC: 277},
		{ID: 702, Type: 7, SubType: "7-4", FinishNPC: 277},
		{ID: 703, Type: 7, SubType: "7-16", FinishNPC: 3},
	}
	manager := newLoopTestManager(t, nil, children)

	requires := []json.RawMessage{
		mustQuestJSON(t, models.QuestRequireTemplate{ID: 1, ItemID: 9001, Kind: 2, Num: 1, Q: 1, Qid: 701, Type: 12}),
		mustQuestJSON(t, models.QuestRequireTemplate{ID: 2, ItemID: 9003, Kind: 2, Num: 1, Q: 1, Qid: 703, Type: 12}),
	}
	if err := manager.GetCache().LoadTable(models.TableQuestRequire, requires); err != nil {
		t.Fatalf("load quest require table: %v", err)
	}
	spawns := []json.RawMessage{
		mustQuestJSON(t, models.MapCreatureTemplate{ID: 1, Cid: 9002, Mid: 5}),
	}
	if err := manager.GetCache().LoadTable(models.TableMapCreature, spawns); err != nil {
		t.Fatalf("load map creature table: %v", err)
	}

	charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 60}}
	svc := NewService(newQuestTestQuestRepo(), logger)
	svc.SetGameDataManager(manager)
	svc.SetCharacterRepository(charRepo)

	for i := 0; i < 40; i++ {
		child := svc.pickNextChild(context.Background(), 1, 4, 0)
		if child == nil {
			t.Fatalf("expected a picked child, got nil")
		}
		if int(child.ID) == 701 {
			t.Fatalf("pickNextChild returned boss-kill round 701 (creature 9001 has no map spawn) while fightable 702 existed")
		}
	}

	child := svc.pickNextChild(context.Background(), 1, 16, 0)
	if child == nil || int(child.ID) != 703 {
		t.Fatalf("expected all-boss pool to fall back to raw pool and return 703, got %v", child)
	}
}

func TestCompleteQuest_LoopChild_ContinuesCycleWithoutHistory(t *testing.T) {
	logger := zap.NewNop()

	loopTpl := models.QuestLoopTemplate{ID: 4, Nid: 277, Num: 2, Refresh: 600}
	children := []models.QuestTemplate{
		{ID: 701, Type: 7, SubType: "7-4", FinishNPC: 277},
		{ID: 702, Type: 7, SubType: "7-4", FinishNPC: 277},
	}
	manager := newLoopTestManager(t, []models.QuestLoopTemplate{loopTpl}, children)

	charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 30}}
	itemService := appitem.NewService(newQuestTestItemRepo(), logger)
	itemService.SetGameDataManager(manager)
	questRepo := newQuestTestQuestRepo()

	progress := domainquest.NewQuestProgress(1, 701, loadTestCompletedKill(t, manager, 701))
	if err := questRepo.Save(context.Background(), progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	loopRepo := newFakeLoopRepo()
	loopRepo.states[1] = map[int]*domainquest.LoopState{
		4: {CharacterID: 1, LoopID: 4, Active: true, TakeDateMs: 12345, Ft: 0, ActiveQuestID: 701},
	}

	svc := NewService(questRepo, logger)
	svc.SetGameDataManager(manager)
	svc.SetCharacterRepository(charRepo)
	svc.SetItemService(itemService)
	svc.SetLoopRepository(loopRepo)

	_, rewards, err := svc.CompleteQuest(context.Background(), 1, 701)
	if err != nil {
		t.Fatalf("CompleteQuest: %v", err)
	}

	if _, err := questRepo.FindByCharacterAndQuest(context.Background(), 1, 701); !errors.Is(err, pkgerrors.ErrNotFound) {
		t.Fatalf("expected finished loop child progress to be deleted, got err=%v", err)
	}
	if len(questRepo.histories) != 0 {
		t.Fatalf("expected no quest_history row for a loop child, got %d", len(questRepo.histories))
	}

	loopInfo, ok := rewards["loop"].(map[string]interface{})
	if !ok || loopInfo["id"] != 4 || loopInfo["ft"] != 1 {
		t.Fatalf("expected rewards[loop] = {id:4, ft:1}, got %#v", rewards["loop"])
	}

	nextQP, ok := rewards["loopNextQuest"].(*domainquest.QuestProgress)
	if !ok || nextQP == nil {
		t.Fatalf("expected a granted next-round quest progress, got %#v", rewards["loopNextQuest"])
	}
	if nextQP.QuestID != 702 {
		t.Fatalf("expected next round to be the only remaining pool child (702), got %d", nextQP.QuestID)
	}

	state := loopRepo.states[1][4]
	if !state.Active || state.Ft != 1 || state.ActiveQuestID != 702 {
		t.Fatalf("expected loop state {active:true, ft:1, activeQuestId:702}, got %+v", state)
	}
}

func TestCompleteQuest_LoopChild_DeactivatesOnFinalRound(t *testing.T) {
	logger := zap.NewNop()

	loopTpl := models.QuestLoopTemplate{ID: 5, Nid: 280, Num: 1, Refresh: 600}
	children := []models.QuestTemplate{
		{ID: 801, Type: 7, SubType: "7-5", FinishNPC: 280},
	}
	manager := newLoopTestManager(t, []models.QuestLoopTemplate{loopTpl}, children)

	charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 50}}
	itemService := appitem.NewService(newQuestTestItemRepo(), logger)
	itemService.SetGameDataManager(manager)
	questRepo := newQuestTestQuestRepo()

	progress := domainquest.NewQuestProgress(1, 801, loadTestCompletedKill(t, manager, 801))
	if err := questRepo.Save(context.Background(), progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	loopRepo := newFakeLoopRepo()
	loopRepo.states[1] = map[int]*domainquest.LoopState{
		5: {CharacterID: 1, LoopID: 5, Active: true, TakeDateMs: 12345, Ft: 0, ActiveQuestID: 801},
	}

	svc := NewService(questRepo, logger)
	svc.SetGameDataManager(manager)
	svc.SetCharacterRepository(charRepo)
	svc.SetItemService(itemService)
	svc.SetLoopRepository(loopRepo)

	_, rewards, err := svc.CompleteQuest(context.Background(), 1, 801)
	if err != nil {
		t.Fatalf("CompleteQuest: %v", err)
	}

	if nextQP, ok := rewards["loopNextQuest"].(*domainquest.QuestProgress); ok && nextQP != nil {
		t.Fatalf("expected no next round on final cycle completion, got %+v", nextQP)
	}

	state := loopRepo.states[1][5]
	if state.Active {
		t.Fatalf("expected loop instance to deactivate once ft reaches num, got active=true")
	}
	if state.TakeDateMs != 12345 {
		t.Fatalf("expected take_date_ms to stay anchored at the original take time, got %d", state.TakeDateMs)
	}
}

func TestCompleteQuest_GuildLoopFailsBeforeProgressOrRoundMutation(t *testing.T) {
	logger := zap.NewNop()
	loopTpl := models.QuestLoopTemplate{ID: loopGuildLid, Nid: 1069, Num: 15, Refresh: 600}
	children := []models.QuestTemplate{{ID: 30001, Type: 7, SubType: "7-2", FinishNPC: 1069}}
	manager := newLoopTestManager(t, []models.QuestLoopTemplate{loopTpl}, children)
	questRepo := newQuestTestQuestRepo()
	progress := domainquest.NewQuestProgress(1, 30001, loadTestCompletedKill(t, manager, 30001))
	if err := questRepo.Save(context.Background(), progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	loopRepo := newFakeLoopRepo()
	loopRepo.states[1] = map[int]*domainquest.LoopState{
		loopGuildLid: {CharacterID: 1, LoopID: loopGuildLid, Active: true, TakeDateMs: 12345, Ft: 4, ActiveQuestID: 30001},
	}
	svc := NewService(questRepo, logger)
	svc.SetGameDataManager(manager)
	svc.SetLoopRepository(loopRepo)

	if _, _, err := svc.CompleteQuest(context.Background(), 1, 30001); !errors.Is(err, errGuildLoopRewardsUnavailable) {
		t.Fatalf("CompleteQuest error = %v, want %v", err, errGuildLoopRewardsUnavailable)
	}
	if _, err := questRepo.FindByCharacterAndQuest(context.Background(), 1, 30001); err != nil {
		t.Fatalf("guild loop progress changed on blocked completion: %v", err)
	}
	state := loopRepo.states[1][loopGuildLid]
	if !state.Active || state.Ft != 4 || state.ActiveQuestID != 30001 {
		t.Fatalf("guild loop state changed on blocked completion: %+v", state)
	}
	if len(questRepo.histories) != 0 {
		t.Fatalf("guild loop wrote history on blocked completion: %d", len(questRepo.histories))
	}
}

func TestCompleteQuest_GuildLoopUsesAtomicNEOCommand(t *testing.T) {
	logger := zap.NewNop()
	loopTpl := models.QuestLoopTemplate{ID: loopGuildLid, Nid: 1069, Num: 15, Refresh: 600}
	children := []models.QuestTemplate{
		{ID: 39001, Type: 7, SubType: "7-2", FinishNPC: 1069},
		{ID: 39002, Type: 7, SubType: "7-2", FinishNPC: 1069},
	}
	manager := newLoopTestManager(t, []models.QuestLoopTemplate{loopTpl}, children)
	requires := []json.RawMessage{mustQuestJSON(t, models.QuestRequireTemplate{ID: 1, ItemID: 500, Kind: domainquest.RequireKindItem, Num: 2, Qid: 39001, Type: domainquest.TableIDItemTemplate})}
	if err := manager.GetCache().LoadTable(models.TableQuestRequire, requires); err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{mustQuestJSON(t, models.ItemTemplateTemplate{ID: 500, Name: "Guild material", StackMax: 99})}); err != nil {
		t.Fatal(err)
	}
	questRepo := newQuestTestQuestRepo()
	objective := domainquest.Objective{Type: domainquest.ObjectiveCollectItem, Target: 500, Current: 2, Required: 2, ItemTableType: domainquest.TableIDItemTemplate}
	progress := domainquest.NewQuestProgress(1, 39001, []domainquest.Objective{objective})
	if err := questRepo.Save(context.Background(), progress); err != nil {
		t.Fatal(err)
	}
	itemRepo := newQuestTestItemRepo()
	item := domainitem.NewItem(1, 500, domainitem.ItemTypeQuest, domainitem.SlotTypeQuestBag, 0)
	item.StackCount = 3
	if err := itemRepo.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	itemService := appitem.NewService(itemRepo, logger)
	itemService.SetGameDataManager(manager)
	loopRepo := &fakeGuildLoopCommandRepo{fakeLoopRepo: newFakeLoopRepo()}
	loopRepo.states[1] = map[int]*domainquest.LoopState{
		loopGuildLid: {CharacterID: 1, LoopID: loopGuildLid, Active: true, Ft: 0, ActiveQuestID: 39001},
	}
	svc := NewService(questRepo, logger)
	svc.SetGameDataManager(manager)
	svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 90}})
	svc.SetItemService(itemService)
	svc.SetLoopRepository(loopRepo)
	loaded, loadErr := questRepo.FindByCharacterAndQuest(context.Background(), 1, 39001)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if syncErr := svc.syncCollectObjectives(context.Background(), loaded); syncErr != nil {
		t.Fatal(syncErr)
	}
	if !loaded.CanComplete() {
		t.Fatalf("test objective not complete after sync: status=%d blocked=%v expired=%v objectives=%+v", loaded.Status, loaded.CompletionBlocked, loaded.IsExpired(), loaded.Objectives)
	}

	_, rewards, err := svc.CompleteQuest(context.Background(), 1, 39001)
	if err != nil {
		t.Fatalf("CompleteQuest: %v", err)
	}
	command := loopRepo.command
	if command.Money != 233280 || command.MoneyBind != 233280 || command.Contribution != 1304 || command.GuildExp != 2608 {
		t.Fatalf("unexpected NEO reward command: %+v", command)
	}
	if len(command.Items) != 1 || command.Items[0].ItemID != item.ID || command.Items[0].Count != 2 {
		t.Fatalf("unexpected item consumption: %+v", command.Items)
	}
	if command.PetIDs == nil || len(command.PetIDs) != 0 {
		t.Fatalf("expected canonical empty pet consumption, got %#v", command.PetIDs)
	}
	if command.NextQuest == nil || command.NextQuest.QuestID != 39002 {
		t.Fatalf("unexpected next quest: %+v", command.NextQuest)
	}
	if next, ok := rewards["loopNextQuest"].(*domainquest.QuestProgress); !ok || next == nil || next.ID != 999 {
		t.Fatalf("unexpected committed next quest reward: %#v", rewards["loopNextQuest"])
	}
	if itemRepo.items[item.ID].StackCount != 3 {
		t.Fatalf("planning mutated item repository before atomic command: %d", itemRepo.items[item.ID].StackCount)
	}
}

func TestPlanGuildLoopConsumptionUsesCanonicalEmptyCollections(t *testing.T) {
	svc := NewService(newQuestTestQuestRepo(), zap.NewNop())
	plan, err := svc.planGuildLoopConsumption(context.Background(), domainquest.NewQuestProgress(1, 39001, nil))
	if err != nil {
		t.Fatalf("planGuildLoopConsumption: %v", err)
	}
	if plan.items == nil || plan.itemPlans == nil || plan.petIDs == nil {
		t.Fatalf("expected non-nil empty collections, got items=%#v itemPlans=%#v petIDs=%#v", plan.items, plan.itemPlans, plan.petIDs)
	}
}

func TestAbandonQuest_Type1DeletesActiveProgress(t *testing.T) {
	logger := zap.NewNop()
	manager := newLoopTestManager(t, nil, []models.QuestTemplate{{ID: 901, Type: predef.QuestTypeNewbie}})
	questRepo := newQuestTestQuestRepo()

	progress := domainquest.NewQuestProgress(1, 901, nil)
	if err := questRepo.Save(context.Background(), progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	svc := NewService(questRepo, logger)
	svc.SetGameDataManager(manager)

	if err := svc.AbandonQuest(context.Background(), 1, 901); err != nil {
		t.Fatalf("AbandonQuest: %v", err)
	}

	if _, err := questRepo.FindByCharacterAndQuest(context.Background(), 1, 901); !errors.Is(err, pkgerrors.ErrNotFound) {
		t.Fatalf("expected type-1 quest progress to be deleted, got err=%v", err)
	}
}

func TestCompleteQuest_DailyQuestAllowsRetakeOnlyAfterToday(t *testing.T) {
	logger := zap.NewNop()
	manager := newLoopTestManager(t, nil, []models.QuestTemplate{{ID: 950, Type: 6, AwardExpRe: 0}})

	charRepo := &questTestCharacterRepo{char: &domainchar.Character{ID: 1, Level: 10}}
	itemService := appitem.NewService(newQuestTestItemRepo(), logger)
	itemService.SetGameDataManager(manager)
	questRepo := newFakeDatedQuestRepo()

	progress := domainquest.NewQuestProgress(1, 950, loadTestCompletedKill(t, manager, 950))
	if err := questRepo.Save(context.Background(), progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	svc := NewService(questRepo, logger)
	svc.SetGameDataManager(manager)
	svc.SetCharacterRepository(charRepo)
	svc.SetItemService(itemService)

	if _, _, err := svc.CompleteQuest(context.Background(), 1, 950); err != nil {
		t.Fatalf("CompleteQuest: %v", err)
	}

	if _, err := questRepo.FindByCharacterAndQuest(context.Background(), 1, 950); !errors.Is(err, pkgerrors.ErrNotFound) {
		t.Fatalf("expected daily quest progress row to be deleted after completion, got err=%v", err)
	}
	if len(questRepo.histories) != 1 {
		t.Fatalf("expected quest_history to record the daily completion, got %d rows", len(questRepo.histories))
	}

	completedToday, err := svc.GetCompletedQuestIDs(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetCompletedQuestIDs: %v", err)
	}
	if !containsInt(completedToday, 950) {
		t.Fatalf("expected daily quest 950 to still gate re-take on completion day, got %v", completedToday)
	}

	questRepo.dates[950] = time.Now().Add(-48 * time.Hour)
	completedLater, err := svc.GetCompletedQuestIDs(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetCompletedQuestIDs: %v", err)
	}
	if containsInt(completedLater, 950) {
		t.Fatalf("expected daily quest 950 completed two days ago to no longer gate re-take, got %v", completedLater)
	}
}

func containsInt(list []int, target int) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
