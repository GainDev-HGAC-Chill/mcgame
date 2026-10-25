// Open-sourced by BaoLT

package achievement

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"

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
