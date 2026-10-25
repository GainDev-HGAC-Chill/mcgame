// Open-sourced by BaoLT

package achievement

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	domaincombat "mcgame-server/internal/domain/combat"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type achievementTestRepo struct {
	rows         []*domainachievement.Progress
	requirements []*domainachievement.RequirementProgress
	claimedIDs   []int64
	reserved     bool
	events       []achievementTestEvent
	facts        []domainachievement.Fact
	processed    []string
}

type achievementTestEvent struct {
	key     string
	updates []domainachievement.RequirementUpdate
}

func (r *achievementTestRepo) GetProgressByCharacter(ctx context.Context, characterID int64) ([]*domainachievement.Progress, error) {
	return r.rows, nil
}

func (r *achievementTestRepo) UpsertProgress(ctx context.Context, progress *domainachievement.Progress) (*domainachievement.Progress, error) {
	return progress, nil
}

func (r *achievementTestRepo) MarkCompleted(ctx context.Context, characterID, achievementID int64, completedAt time.Time) (*domainachievement.Progress, error) {
	return nil, nil
}

func (r *achievementTestRepo) MarkClaimed(ctx context.Context, characterID, achievementID int64, claimedAt time.Time) (*domainachievement.Progress, error) {
	r.claimedIDs = append(r.claimedIDs, achievementID)
	for _, row := range r.rows {
		if row == nil || row.AchievementID != achievementID {
			continue
		}
		row.IsClaimed = true
		row.ClaimedAt = &claimedAt
		return row, nil
	}
	return nil, nil
}

func (r *achievementTestRepo) ListClaimedCounts(ctx context.Context, characterID int64) (map[int64]int, error) {
	return nil, nil
}

func (r *achievementTestRepo) ListRequirementProgress(ctx context.Context, characterID int64) ([]*domainachievement.RequirementProgress, error) {
	return r.requirements, nil
}

func (r *achievementTestRepo) RecordEvent(ctx context.Context, characterID int64, eventKey string, updates []domainachievement.RequirementUpdate, occurredAt time.Time) (domainachievement.ProgressUpdate, error) {
	r.events = append(r.events, achievementTestEvent{key: eventKey, updates: updates})
	return domainachievement.ProgressUpdate{
		Requirements: []*domainachievement.RequirementProgress{},
		Completed:    []domainachievement.CompletionEntry{},
	}, nil
}

func (r *achievementTestRepo) RecordLoginDay(ctx context.Context, characterID int64, updates []domainachievement.RequirementUpdate, occurredAt time.Time) (domainachievement.ProgressUpdate, error) {
	r.events = append(r.events, achievementTestEvent{key: "login-day:v2", updates: updates})
	return domainachievement.ProgressUpdate{
		Requirements: []*domainachievement.RequirementProgress{},
		Completed:    []domainachievement.CompletionEntry{},
	}, nil
}

func (r *achievementTestRepo) ListPendingFacts(ctx context.Context, characterID int64, limit int) ([]domainachievement.Fact, error) {
	return append([]domainachievement.Fact(nil), r.facts...), nil
}

func (r *achievementTestRepo) MarkFactProcessed(ctx context.Context, factID string, processedAt time.Time) (bool, error) {
	r.processed = append(r.processed, factID)
	return true, nil
}

func (r *achievementTestRepo) ReserveClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID, reservedAt time.Time) (bool, error) {
	if r.reserved {
		return false, nil
	}
	for _, row := range r.rows {
		if row != nil && row.AchievementID == achievementID && row.IsCompleted && !row.IsClaimed {
			r.reserved = true
			return true, nil
		}
	}
	return false, nil
}

func (r *achievementTestRepo) FinalizeClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID, claimedAt time.Time) (*domainachievement.Progress, error) {
	if !r.reserved {
		return nil, nil
	}
	for _, row := range r.rows {
		if row != nil && row.AchievementID == achievementID {
			row.IsClaimed = true
			row.ClaimedAt = &claimedAt
			r.claimedIDs = append(r.claimedIDs, achievementID)
			r.reserved = false
			return row, nil
		}
	}
	return nil, nil
}

func (r *achievementTestRepo) ReleaseClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID) error {
	r.reserved = false
	return nil
}

func TestLoadSnapshotBuildsClientMaps(t *testing.T) {
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	manager := gamedata.NewManager(nil, zap.NewNop())

	achievementRows := []json.RawMessage{
		mustAchievementJSON(t, models.AchievementTemplate{ID: 1001, Award: 5}),
		mustAchievementJSON(t, models.AchievementTemplate{ID: 1002, Award: 7}),
	}
	requirementRows := []json.RawMessage{
		mustAchievementJSON(t, models.AchievementRequireTemplate{ID: 1, Aid: 1001, Num: 10}),
		mustAchievementJSON(t, models.AchievementRequireTemplate{ID: 2, Aid: 1002, Num: 8}),
	}

	if err := manager.GetCache().LoadTable(models.TableAchievement, achievementRows); err != nil {
		t.Fatalf("load achievement table: %v", err)
	}
	if err := manager.GetCache().LoadTable(models.TableAchievementRequire, requirementRows); err != nil {
		t.Fatalf("load achievement require table: %v", err)
	}

	repo := &achievementTestRepo{rows: []*domainachievement.Progress{
		{
			ID:            1,
			CharacterID:   99,
			AchievementID: 1001,
			Progress:      10,
			Target:        0,
			IsCompleted:   true,
			IsClaimed:     true,
			CompletedAt:   &now,
		},
		{
			ID:            2,
			CharacterID:   99,
			AchievementID: 1002,
			Progress:      3,
			Target:        0,
			IsCompleted:   false,
			IsClaimed:     false,
		},
	}, requirements: []*domainachievement.RequirementProgress{
		{CharacterID: 99, RequirementID: 1, AchievementID: 1001, Progress: 10, Target: 10, Done: true},
		{CharacterID: 99, RequirementID: 2, AchievementID: 1002, Progress: 3, Target: 8, Done: false},
	}}

	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)

	snapshot, err := service.LoadSnapshot(context.Background(), 99)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v", err)
	}

	if got := snapshot.AchievementPoints; got != 5 {
		t.Fatalf("AchievementPoints = %d, want 5", got)
	}
	if got := snapshot.AchieveLog["1001"]; got != now.UnixMilli() {
		t.Fatalf("AchieveLog[1001] = %v, want %d", got, now.UnixMilli())
	}
	if got := snapshot.TakeAchieveAwardLog["1001"]; got != 1 {
		t.Fatalf("TakeAchieveAwardLog[1001] = %v, want 1", got)
	}

	progressEntry, ok := snapshot.AchieveReqLog["2"].(map[string]interface{})
	if !ok {
		t.Fatalf("AchieveReqLog[2] type = %T, want map[string]interface{}", snapshot.AchieveReqLog["2"])
	}
	if got := progressEntry["done"]; got != 0 {
		t.Fatalf("AchieveReqLog[2][done] = %v, want 0", got)
	}
	if got := progressEntry["progress"]; got != 3 {
		t.Fatalf("AchieveReqLog[2][progress] = %v, want 3", got)
	}
}

func TestClaimReservationFinalizesCompletedAchievement(t *testing.T) {
	repo := &achievementTestRepo{rows: []*domainachievement.Progress{{
		ID:            1,
		CharacterID:   99,
		AchievementID: 1008,
		Progress:      1,
		Target:        1,
		IsCompleted:   true,
		IsClaimed:     false,
	}}}

	service := NewService(repo, zap.NewNop())
	token := uuid.New()
	if err := service.ReserveClaim(context.Background(), 99, 1008, token); err != nil {
		t.Fatalf("ReserveClaim() error = %v", err)
	}
	if err := service.FinalizeClaim(context.Background(), 99, 1008, token); err != nil {
		t.Fatalf("FinalizeClaim() error = %v", err)
	}
	if len(repo.claimedIDs) != 1 || repo.claimedIDs[0] != 1008 {
		t.Fatalf("claimedIDs = %#v, want [1008]", repo.claimedIDs)
	}
	if !repo.rows[0].IsClaimed {
		t.Fatalf("progress row should be marked claimed")
	}
}

func TestClaimReservationRejectsAlreadyClaimedAchievement(t *testing.T) {
	repo := &achievementTestRepo{rows: []*domainachievement.Progress{{
		ID:            1,
		CharacterID:   99,
		AchievementID: 1021,
		Progress:      1,
		Target:        1,
		IsCompleted:   true,
		IsClaimed:     true,
	}}}

	service := NewService(repo, zap.NewNop())
	err := service.ReserveClaim(context.Background(), 99, 1021, uuid.New())
	if err == nil {
		t.Fatalf("ReserveClaim() error = nil, want already claimed error")
	}
	if len(repo.claimedIDs) != 0 {
		t.Fatalf("claimedIDs = %#v, want empty", repo.claimedIDs)
	}
}

func TestRecordLoginUsesCanonicalLevelAndDailyRequirements(t *testing.T) {
	repo := &achievementTestRepo{}
	service := NewService(repo, zap.NewNop())
	now := time.Date(2026, 10, 19, 9, 30, 0, 0, time.UTC)

	if _, err := service.RecordLogin(context.Background(), 99, 52, now); err != nil {
		t.Fatalf("RecordLogin() error = %v", err)
	}
	if len(repo.events) != 2 {
		t.Fatalf("events = %d, want 2", len(repo.events))
	}
	if repo.events[0].key != "level:v1:52" {
		t.Fatalf("level event key = %q", repo.events[0].key)
	}
	if len(repo.events[0].updates) != 4 || repo.events[0].updates[1].RequirementID != 2 || repo.events[0].updates[1].Absolute == nil || *repo.events[0].updates[1].Absolute != 52 {
		t.Fatalf("level updates = %#v", repo.events[0].updates)
	}
	wantTargets := []int{10, 50, 100, 140}
	for index, update := range repo.events[0].updates {
		if update.Target != wantTargets[index] {
			t.Fatalf("level update %d target = %d, want %d", index, update.Target, wantTargets[index])
		}
	}
	if repo.events[1].key != "login-day:v2" {
		t.Fatalf("login event key = %q", repo.events[1].key)
	}
	if len(repo.events[1].updates) != 6 || repo.events[1].updates[0].RequirementID != 53 || repo.events[1].updates[5].RequirementID != 424 {
		t.Fatalf("login updates = %#v", repo.events[1].updates)
	}
}

func TestRecordBattleEndedAddsArenaRequirementsOnlyForWins(t *testing.T) {
	repo := &achievementTestRepo{}
	service := NewService(repo, zap.NewNop())
	now := time.Date(2026, 10, 19, 9, 30, 0, 0, time.UTC)
	battleID := "4aa1a819-0ca6-4f8f-a52c-f762ba96f2ee"

	if _, err := service.RecordBattleEnded(context.Background(), 99, battleID, 3, true, now); err != nil {
		t.Fatalf("RecordBattleEnded() error = %v", err)
	}
	if len(repo.events) != 1 || repo.events[0].key != "battle:v1:"+battleID {
		t.Fatalf("events = %#v", repo.events)
	}
	if len(repo.events[0].updates) != 9 || repo.events[0].updates[5].RequirementID != 151 {
		t.Fatalf("arena updates = %#v", repo.events[0].updates)
	}
	if _, err := service.RecordBattleEnded(context.Background(), 99, "cb22f355-06a1-451c-a22a-cbca876ff0d4", 3, false, now); err != nil {
		t.Fatalf("lost RecordBattleEnded() error = %v", err)
	}
	if len(repo.events) != 1 {
		t.Fatalf("lost battle recorded an event")
	}
}

func TestProgressEvaluatorsRejectMissingBusinessAuthority(t *testing.T) {
	repo := &achievementTestRepo{}
	service := NewService(repo, zap.NewNop())
	now := time.Date(2026, 10, 26, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "login character", run: func() error { _, err := service.RecordLogin(context.Background(), 0, 100, now); return err }},
		{name: "login time", run: func() error { _, err := service.RecordLogin(context.Background(), 99, 100, time.Time{}); return err }},
		{name: "level", run: func() error { _, err := service.RecordLevel(context.Background(), 99, 0, now); return err }},
		{name: "quest progress id", run: func() error {
			_, err := service.RecordQuestFinished(context.Background(), 99, 10, 0, 1, now)
			return err
		}},
		{name: "quest completion count", run: func() error {
			_, err := service.RecordQuestFinished(context.Background(), 99, 10, 20, 0, now)
			return err
		}},
		{name: "battle id", run: func() error {
			_, err := service.RecordBattleEnded(context.Background(), 99, " ", domaincombat.BattleTypePVE, true, now)
			return err
		}},
		{name: "battle type", run: func() error {
			_, err := service.RecordBattleEnded(context.Background(), 99, "4aa1a819-0ca6-4f8f-a52c-f762ba96f2ee", domaincombat.BattleType(99), true, now)
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				t.Fatal("event error = nil, want fail-closed rejection")
			}
		})
	}
	if len(repo.events) != 0 {
		t.Fatalf("invalid events were persisted: %#v", repo.events)
	}
}

func TestOfficialRewardRegistry(t *testing.T) {
	expected := map[int64]RewardDefinition{
		1008: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 2},
		1015: {Kind: RewardKindMedal, TemplateID: 449, Quantity: 1},
		1021: {Kind: RewardKindTitle, TemplateID: 277, Quantity: 1},
		1032: {Kind: RewardKindMedal, TemplateID: 328, Quantity: 1},
		1035: {Kind: RewardKindMedal, TemplateID: 361, Quantity: 1},
		2012: {Kind: RewardKindMedal, TemplateID: 111, Quantity: 1},
		2013: {Kind: RewardKindTitle, TemplateID: 257, Quantity: 1},
		2015: {Kind: RewardKindMedal, TemplateID: 23, Quantity: 1},
		2017: {Kind: RewardKindTitle, TemplateID: 256, Quantity: 1},
		2019: {Kind: RewardKindMedal, TemplateID: 1, Quantity: 1},
		2032: {Kind: RewardKindMedal, TemplateID: 155, Quantity: 1},
		4004: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 2},
		4013: {Kind: RewardKindTitle, TemplateID: 275, Quantity: 1},
		5050: {Kind: RewardKindTitle, TemplateID: 279, Quantity: 1},
		6026: {Kind: RewardKindMedal, TemplateID: 394, Quantity: 1},
		6030: {Kind: RewardKindTitle, TemplateID: 276, Quantity: 1},
		6078: {Kind: RewardKindTitle, TemplateID: 278, Quantity: 1},
		6092: {Kind: RewardKindTitle, TemplateID: 273, Quantity: 1},
		6096: {Kind: RewardKindMedal, TemplateID: 383, Quantity: 1},
		6102: {Kind: RewardKindMedal, TemplateID: 416, Quantity: 1},
		6103: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 2},
		6150: {Kind: RewardKindMedal, TemplateID: 317, Quantity: 1},
		6153: {Kind: RewardKindMedal, TemplateID: 133, Quantity: 1},
		6170: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 3},
		6172: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 2},
		6187: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 1},
		6190: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 1},
		6191: {Kind: RewardKindItem, TemplateID: 3984, Quantity: 3},
		6192: {Kind: RewardKindItem, TemplateID: 4029, Quantity: 5},
	}

	if len(achievementRewards) != len(expected) {
		t.Fatalf("reward count = %d, want %d", len(achievementRewards), len(expected))
	}
	for achievementID, want := range expected {
		got, ok := RewardForAchievement(achievementID)
		if !ok || got != want {
			t.Fatalf("reward %d = %#v, %v; want %#v", achievementID, got, ok, want)
		}
	}
	if _, ok := RewardForAchievement(6066); ok {
		t.Fatalf("achievement 6066 must fail closed without canonical title ID")
	}
}

func mustAchievementJSON(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
