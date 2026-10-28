// Open-sourced by BaoLT

// Achievement fact processing evaluates durable business events idempotently.
package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	domainpet "mcgame-server/internal/domain/pet"

	"github.com/google/uuid"
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
		updates, evaluateErr := s.evaluateAchievementFact(fact)
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

func (s *Service) evaluateAchievementFact(fact domainachievement.Fact) ([]domainachievement.RequirementUpdate, error) {
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
		updates := buildIncrementUpdates(questRequirements)
		specific, err := s.specificQuestFactUpdates(payload.QuestID)
		if err != nil {
			return nil, err
		}
		return append(updates, specific...), nil
	case domainachievement.FactKindPetCaptured:
		var payload domainachievement.PetCapturedFactPayload
		if err := json.Unmarshal(fact.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode achievement capture fact %q: %w", fact.FactID, err)
		}
		if payload.PetID <= 0 || payload.CreatureTemplateID <= 0 || fact.FactID != fmt.Sprintf("pet-captured:v1:%d", payload.PetID) {
			return nil, fmt.Errorf("invalid achievement capture fact %q", fact.FactID)
		}
		return buildIncrementUpdates([]requirementTarget{
			{RequirementID: 83, AchievementID: 2016, Target: 1},
			{RequirementID: 84, AchievementID: 2017, Target: 5000},
		}), nil
	case domainachievement.FactKindWishPetDrawn:
		var payload domainachievement.WishPetDrawnFactPayload
		if err := json.Unmarshal(fact.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode achievement Wish Pet fact %q: %w", fact.FactID, err)
		}
		drawID, err := uuid.Parse(payload.DrawID)
		expectedID := fmt.Sprintf("wish-pet-drawn:v1:%s:%d", payload.DrawID, payload.Ordinal)
		if err != nil || drawID.String() != payload.DrawID || payload.Ordinal < 0 || payload.PlanID <= 0 || payload.CreatureTemplateID <= 0 || payload.GrowRate <= 0 || payload.ColorTier < 0 || payload.ColorTier > 4 || domainpet.ColorByGrowRate(payload.GrowRate) != payload.ColorTier || fact.FactID != expectedID {
			return nil, fmt.Errorf("invalid achievement Wish Pet fact %q", fact.FactID)
		}
		requirements := []requirementTarget{{RequirementID: 106, AchievementID: 2039, Target: 1}}
		switch payload.ColorTier {
		case 2:
			requirements = append(requirements, requirementTarget{RequirementID: 107, AchievementID: 2040, Target: 1})
		case 3:
			requirements = append(requirements, requirementTarget{RequirementID: 108, AchievementID: 2041, Target: 1})
		case 4:
			requirements = append(requirements, requirementTarget{RequirementID: 109, AchievementID: 2042, Target: 1})
		}
		return buildIncrementUpdates(requirements), nil
	case domainachievement.FactKindCharacterRenamed:
		var payload domainachievement.CharacterRenamedFactPayload
		if err := json.Unmarshal(fact.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode achievement character rename fact %q: %w", fact.FactID, err)
		}
		operationID, err := uuid.Parse(payload.OperationID)
		expectedID := "character-renamed:v1:" + payload.OperationID
		if err != nil || operationID.String() != payload.OperationID || payload.CardItemID <= 0 || payload.NewName == "" || fact.FactID != expectedID {
			return nil, fmt.Errorf("invalid achievement character rename fact %q", fact.FactID)
		}
		return buildIncrementUpdates([]requirementTarget{{RequirementID: 150, AchievementID: 3040, Target: 1}}), nil
	default:
		return nil, fmt.Errorf("unsupported achievement fact kind %q", fact.Kind)
	}
}

func (s *Service) specificQuestFactUpdates(questID int) ([]domainachievement.RequirementUpdate, error) {
	if s == nil || s.gameDataManager == nil {
		return nil, nil
	}
	template := s.gameDataManager.GetQuest(questID)
	switch questID {
	case 2579:
		if template == nil || template.Name != "Tiêu Diệt Phản Quân Đại Tướng" || int(template.Type) != 6 || template.SubType != "-1" {
			return nil, fmt.Errorf("quest authority for requirement 226 is invalid")
		}
		return buildIncrementUpdates([]requirementTarget{{RequirementID: 226, AchievementID: 5031, Target: 1}}), nil
	case 3202:
		if template == nil || template.Name != "Làm 1 Nhiệm Vụ Treo Thưởng" || int(template.Type) != 1 || template.SubType != "-1" {
			return nil, fmt.Errorf("quest authority for requirement 254 is invalid")
		}
		return buildIncrementUpdates([]requirementTarget{{RequirementID: 254, AchievementID: 5048, Target: 1}}), nil
	default:
		return nil, nil
	}
}
