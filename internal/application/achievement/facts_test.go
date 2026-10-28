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

func TestProcessPendingFactsRecordsCapturedPetOnce(t *testing.T) {
	now := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	repo := &achievementTestRepo{facts: []domainachievement.Fact{{
		FactID:      "pet-captured:v1:7001",
		CharacterID: 99,
		Kind:        domainachievement.FactKindPetCaptured,
		Payload:     json.RawMessage(`{"pet_id":7001,"creature_template_id":516}`),
		OccurredAt:  now,
	}}}
	service := NewService(repo, zap.NewNop())
	if _, err := service.ProcessPendingFacts(context.Background(), 99, now.Add(time.Minute)); err != nil {
		t.Fatalf("ProcessPendingFacts() error = %v", err)
	}
	if len(repo.events) != 1 || len(repo.events[0].updates) != 2 || repo.events[0].updates[0].RequirementID != 83 || repo.events[0].updates[0].Target != 1 || repo.events[0].updates[1].RequirementID != 84 || repo.events[0].updates[1].AchievementID != 2017 || repo.events[0].updates[1].Target != 5000 {
		t.Fatalf("capture updates = %#v", repo.events)
	}
	if len(repo.processed) != 1 {
		t.Fatalf("processed = %#v", repo.processed)
	}
}

func TestProcessPendingFactsMapsWishPetColor(t *testing.T) {
	now := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		growRate    float64
		colorTier   int
		requirement int64
	}{
		{name: "green", growRate: 1.2, colorTier: 1},
		{name: "blue", growRate: 1.5, colorTier: 2, requirement: 107},
		{name: "purple", growRate: 1.9, colorTier: 3, requirement: 108},
		{name: "orange", growRate: 2.3, colorTier: 4, requirement: 109},
	}
	for ordinal, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(domainachievement.WishPetDrawnFactPayload{
				DrawID:             "8a89bb8f-9323-4f58-96dc-035fcdadf479",
				Ordinal:            ordinal,
				PlanID:             int64(100 + ordinal),
				CreatureTemplateID: 516,
				GrowRate:           test.growRate,
				ColorTier:          test.colorTier,
			})
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			repo := &achievementTestRepo{facts: []domainachievement.Fact{{
				FactID:      "wish-pet-drawn:v1:8a89bb8f-9323-4f58-96dc-035fcdadf479:" + string(rune('0'+ordinal)),
				CharacterID: 99,
				Kind:        domainachievement.FactKindWishPetDrawn,
				Payload:     payload,
				OccurredAt:  now,
			}}}
			service := NewService(repo, zap.NewNop())
			if _, err := service.ProcessPendingFacts(context.Background(), 99, now.Add(time.Minute)); err != nil {
				t.Fatalf("ProcessPendingFacts() error = %v", err)
			}
			updates := repo.events[0].updates
			if len(updates) == 0 || updates[0].RequirementID != 106 {
				t.Fatalf("Wish updates = %#v", updates)
			}
			if test.requirement == 0 && len(updates) != 1 {
				t.Fatalf("green Wish updates = %#v", updates)
			}
			if test.requirement != 0 && (len(updates) != 2 || updates[1].RequirementID != test.requirement) {
				t.Fatalf("colored Wish updates = %#v", updates)
			}
		})
	}
}

func TestProcessPendingFactsRejectsWishColorMismatch(t *testing.T) {
	now := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	repo := &achievementTestRepo{facts: []domainachievement.Fact{{
		FactID:      "wish-pet-drawn:v1:8a89bb8f-9323-4f58-96dc-035fcdadf479:0",
		CharacterID: 99,
		Kind:        domainachievement.FactKindWishPetDrawn,
		Payload:     json.RawMessage(`{"draw_id":"8a89bb8f-9323-4f58-96dc-035fcdadf479","ordinal":0,"plan_id":101,"creature_template_id":516,"grow_rate":1.5,"color_tier":4}`),
		OccurredAt:  now,
	}}}
	service := NewService(repo, zap.NewNop())
	if _, err := service.ProcessPendingFacts(context.Background(), 99, now); err == nil {
		t.Fatal("ProcessPendingFacts() error = nil, want color mismatch rejection")
	}
	if len(repo.events) != 0 || len(repo.processed) != 0 {
		t.Fatalf("invalid Wish fact mutated state: events=%#v processed=%#v", repo.events, repo.processed)
	}
}

func TestProcessPendingFactsRecordsCharacterRenameOnce(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	repo := &achievementTestRepo{facts: []domainachievement.Fact{{
		FactID:      "character-renamed:v1:8a89bb8f-9323-4f58-96dc-035fcdadf479",
		CharacterID: 99,
		Kind:        domainachievement.FactKindCharacterRenamed,
		Payload:     json.RawMessage(`{"operation_id":"8a89bb8f-9323-4f58-96dc-035fcdadf479","card_item_id":7001,"new_name":"Tên Mới"}`),
		OccurredAt:  now,
	}}}
	service := NewService(repo, zap.NewNop())
	if _, err := service.ProcessPendingFacts(context.Background(), 99, now.Add(time.Minute)); err != nil {
		t.Fatalf("ProcessPendingFacts() error = %v", err)
	}
	if len(repo.events) != 1 || len(repo.events[0].updates) != 1 || repo.events[0].updates[0].RequirementID != 150 {
		t.Fatalf("rename updates = %#v", repo.events)
	}
	if len(repo.processed) != 1 {
		t.Fatalf("processed = %#v", repo.processed)
	}
}
