// Special script-driven quests must not inherit empty-objective completion.
// CompletionBlocked is runtime verification state, not a persisted objective or reward.
package quest

import (
	"context"

	"mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/social"
)

type FriendRepository interface {
	FindFriends(context.Context, int64) ([]*social.Relationship, error)
}

func (s *Service) SetFriendRepository(repo FriendRepository) {
	s.friendRepository = repo
}

func (s *Service) syncSpecialRequirements(ctx context.Context, qp *quest.QuestProgress) error {
	if qp == nil || qp.Status != quest.QuestStatusActive {
		return nil
	}
	qp.CompletionBlocked = true
	if s.gameDataManager == nil || qp.IsExpired() {
		return nil
	}
	tpl := s.gameDataManager.GetQuest(qp.QuestID)
	if tpl != nil && int(tpl.Type) == 11 {
		if s.guildMembership == nil {
			return nil
		}
		inGuild, err := s.guildMembership.IsInGuild(ctx, qp.CharacterID)
		if err != nil {
			return err
		}
		if !inGuild {
			return nil
		}
	}
	policy, required := ClassifyQuestCompletion(tpl, s.gameDataManager.GetQuestRequire(qp.QuestID))
	switch policy {
	case PolicyCatalog:
		if !s.catalogTargetsAvailable(qp) {
			return nil
		}
		for _, objective := range qp.Objectives {
			if objective.Type == quest.ObjectiveCollectItem && s.itemService == nil {
				return nil
			}
			if objective.Type == quest.ObjectiveSubmitPet && s.petService == nil {
				return nil
			}
		}
		qp.CompletionBlocked = false
		return nil
	case PolicyConversation:
		qp.CompletionBlocked = !s.atQuestNPC(ctx, qp.CharacterID, required)
		return nil
	case PolicyCharacterLevel:
		if s.charRepo == nil {
			return nil
		}
		char, err := s.GetCharacter(ctx, qp.CharacterID)
		if err != nil {
			return err
		}
		qp.CompletionBlocked = char == nil || char.ID != qp.CharacterID || char.Level < required
		return nil
	case PolicyPetLevel, PolicyPetCount:
		if s.petService == nil {
			return nil
		}
		pets, err := s.petService.GetPetList(ctx, qp.CharacterID)
		if err != nil {
			return err
		}
		seen := make(map[int64]bool)
		maxLevel := 0
		for _, pet := range pets {
			if pet == nil || pet.ID <= 0 || pet.CharacterID != qp.CharacterID {
				continue
			}
			seen[pet.ID] = true
			maxLevel = max(maxLevel, pet.Level)
		}
		if policy == PolicyPetCount {
			qp.CompletionBlocked = len(seen) < required
		} else {
			qp.CompletionBlocked = maxLevel < required
		}
		return nil
	case PolicyFriendship:
	default:
		return nil
	}
	if s.friendRepository == nil {
		return nil
	}
	friends, err := s.friendRepository.FindFriends(ctx, qp.CharacterID)
	if err != nil {
		return err
	}
	seen := make(map[int64]bool)
	allIntimate := true
	for _, friend := range friends {
		if friend == nil || friend.CharacterID != qp.CharacterID || friend.OtherID <= 0 || friend.OtherID == qp.CharacterID || friend.Type != social.RelationshipTypeFriend {
			continue
		}
		seen[friend.OtherID] = true
		if friend.Intimacy < 50 {
			allIntimate = false
		}
	}
	qp.CompletionBlocked = len(seen) < required || !allIntimate
	return nil
}
