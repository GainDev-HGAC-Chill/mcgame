// Open-sourced by BaoLT

// Achievement fact processing evaluates durable business events idempotently.
package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
)

const maxAchievementFactBatch = 100

type questCompletedFactPayload struct {
	ProgressID      int64 `json:"progress_id"`
	QuestID         int   `json:"quest_id"`
	CompletionCount int   `json:"completion_count"`
}

func (s *Service) ProcessPendingFacts(ctx context.Context, charID int64, now time.Time) (domainachievement.ProgressUpdate, error) {
	if s.repo == nil {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("achievement repository not configured")
	}
	if charID <= 0 || now.IsZero() {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement fact request")
	}
	facts, err := s.repo.ListPendingFacts(ctx, charID, maxAchievementFactBatch)
	if err != nil {
		return domainachievement.ProgressUpdate{}, err
	}
	results := make([]domainachievement.ProgressUpdate, 0, len(facts))
	for _, fact := range facts {
		if fact.CharacterID != charID {
			return mergeProgressUpdates(results...), fmt.Errorf("achievement fact character mismatch")
		}
		updates, evaluateErr := evaluateAchievementFact(fact)
		if evaluateErr != nil {
			return mergeProgressUpdates(results...), evaluateErr
		}
		result, recordErr := s.recordEvent(ctx, charID, "fact:v1:"+fact.FactID, updates, fact.OccurredAt)
		if recordErr != nil {
			return mergeProgressUpdates(results...), recordErr
		}
		processed, markErr := s.repo.MarkFactProcessed(ctx, fact.FactID, now.UTC())
		if markErr != nil {
			return mergeProgressUpdates(results...), markErr
		}
		if !processed {
			return mergeProgressUpdates(results...), fmt.Errorf("achievement fact %q was not marked processed", fact.FactID)
		}
		results = append(results, result)
	}
	return mergeProgressUpdates(results...), nil
}

func evaluateAchievementFact(fact domainachievement.Fact) ([]domainachievement.RequirementUpdate, error) {
	if fact.FactID == "" || fact.CharacterID <= 0 || fact.OccurredAt.IsZero() {
		return nil, fmt.Errorf("invalid achievement fact identity")
	}
	switch fact.Kind {
	case domainachievement.FactKindQuestCompleted:
		var payload questCompletedFactPayload
		if err := json.Unmarshal(fact.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode achievement quest fact %q: %w", fact.FactID, err)
		}
		if payload.ProgressID <= 0 || payload.QuestID <= 0 || payload.CompletionCount <= 0 {
			return nil, fmt.Errorf("invalid achievement quest fact %q", fact.FactID)
		}
		expectedID := fmt.Sprintf("quest-completed:v1:%d:%d", payload.ProgressID, payload.CompletionCount)
		if fact.FactID != expectedID {
			return nil, fmt.Errorf("achievement quest fact identity mismatch")
		}
		return buildIncrementUpdates(questRequirements), nil
	default:
		return nil, fmt.Errorf("unsupported achievement fact kind %q", fact.Kind)
	}
}
