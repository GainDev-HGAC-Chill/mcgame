// Open-sourced by BaoLT

package achievement

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestProcessPendingFactsRecordsQuestCompletionOnce(t *testing.T) {
	occurredAt := time.Date(2026, 10, 26, 8, 30, 0, 0, time.UTC)
	repo := &achievementTestRepo{facts: []domainachievement.Fact{{
		FactID:      "quest-completed:v1:71:2",
		CharacterID: 99,
		Kind:        domainachievement.FactKindQuestCompleted,
		Payload:     json.RawMessage(`{"progress_id":71,"quest_id":400,"completion_count":2}`),
		OccurredAt:  occurredAt,
	}}}
	service := NewService(repo, zap.NewNop())

	if _, err := service.ProcessPendingFacts(context.Background(), 99, occurredAt.Add(time.Minute)); err != nil {
		t.Fatalf("ProcessPendingFacts() error = %v", err)
	}
	if len(repo.events) != 1 || repo.events[0].key != "fact:v1:quest-completed:v1:71:2" {
		t.Fatalf("events = %#v", repo.events)
	}
	if len(repo.events[0].updates) != 4 || repo.events[0].updates[0].RequirementID != 248 || repo.events[0].updates[3].RequirementID != 251 {
		t.Fatalf("updates = %#v", repo.events[0].updates)
	}
	if len(repo.processed) != 1 || repo.processed[0] != repo.facts[0].FactID {
		t.Fatalf("processed = %#v", repo.processed)
	}
}

func TestProcessPendingFactsMatchesExactQuestAchievement(t *testing.T) {
	now := time.Date(2026, 10, 26, 8, 30, 0, 0, time.UTC)
	repo := &achievementTestRepo{facts: []domainachievement.Fact{{
		FactID:      "quest-completed:v1:81:1",
		CharacterID: 99,
		Kind:        domainachievement.FactKindQuestCompleted,
		Payload:     json.RawMessage(`{"progress_id":81,"quest_id":2579,"completion_count":1}`),
		OccurredAt:  now,
	}}}
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, err := json.Marshal(models.QuestTemplate{ID: 2579, Name: "Tiêu Diệt Phản Quân Đại Tướng", Type: 6, SubType: "-1"})
	if err != nil {
		t.Fatalf("marshal quest: %v", err)
	}
	if err := manager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load quest: %v", err)
	}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)

	if _, err := service.ProcessPendingFacts(context.Background(), 99, now.Add(time.Minute)); err != nil {
		t.Fatalf("ProcessPendingFacts() error = %v", err)
	}
	if len(repo.events) != 1 || len(repo.events[0].updates) != 5 || repo.events[0].updates[4].RequirementID != 226 {
		t.Fatalf("updates = %#v", repo.events)
	}
}

func TestProcessPendingFactsRejectsQuestIdentityMismatch(t *testing.T) {
	now := time.Date(2026, 10, 26, 8, 30, 0, 0, time.UTC)
	repo := &achievementTestRepo{facts: []domainachievement.Fact{{
		FactID:      "quest-completed:v1:71:3",
		CharacterID: 99,
		Kind:        domainachievement.FactKindQuestCompleted,
		Payload:     json.RawMessage(`{"progress_id":71,"quest_id":400,"completion_count":2}`),
		OccurredAt:  now,
	}}}
	service := NewService(repo, zap.NewNop())

	if _, err := service.ProcessPendingFacts(context.Background(), 99, now); err == nil {
		t.Fatal("ProcessPendingFacts() error = nil, want fail-closed rejection")
	}
	if len(repo.events) != 0 || len(repo.processed) != 0 {
		t.Fatalf("invalid fact mutated state: events=%#v processed=%#v", repo.events, repo.processed)
	}
}
