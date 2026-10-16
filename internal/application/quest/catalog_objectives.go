// Rehydrate missing active definitions from the catalog; retain only matching trusted event progress.
package quest

import (
	"mcgame-server/internal/domain/quest"
)

func (s *Service) reconcileCatalogObjectives(qp *quest.QuestProgress) error {
	if qp == nil || qp.Status != quest.QuestStatusActive || s.gameDataManager == nil {
		return nil
	}
	tpl := s.gameDataManager.GetQuest(qp.QuestID)
	policy, _ := ClassifyQuestCompletion(tpl, s.gameDataManager.GetQuestRequire(qp.QuestID))
	if policy == PolicyConversation || policy == PolicyCharacterLevel || policy == PolicyPetLevel || policy == PolicyPetCount || policy == PolicyFriendship || policy == PolicyClassRank {
		qp.Objectives = nil
		return nil
	}
	if policy != PolicyCatalog {
		return nil
	}
	expected := s.GetInitialObjectives(qp.QuestID)
	used := make([]bool, len(qp.Objectives))
	for i := range expected {
		for j, stored := range qp.Objectives {
			if used[j] || !sameObjectiveDefinition(expected[i], stored) {
				continue
			}
			if stored.Current > 0 {
				expected[i].Current = min(stored.Current, expected[i].Required)
			}
			if expected[i].Type == quest.ObjectiveKillMonster {
				expected[i].KillCreditEvents = append([]string(nil), stored.KillCreditEvents...)
			}
			used[j] = true
			break
		}
	}
	qp.Objectives = expected
	return nil
}

func sameObjectiveDefinition(a, b quest.Objective) bool {
	if a.Type != b.Type || a.Target != b.Target || a.Required != b.Required {
		return false
	}
	if a.Type == quest.ObjectiveCollectItem {
		return a.ClientTableType() == b.ClientTableType() && a.Quality == b.Quality
	}
	if a.Type == quest.ObjectiveSubmitPet {
		return a.Quality == b.Quality
	}
	return true
}

func (s *Service) catalogTargetsAvailable(qp *quest.QuestProgress) bool {
	for _, obj := range qp.Objectives {
		switch obj.Type {
		case quest.ObjectiveCollectItem:
			if obj.ClientTableType() == quest.TableIDEquiptTemplate {
				if s.gameDataManager.GetEquipment(obj.Target) == nil {
					return false
				}
			} else if s.gameDataManager.GetItem(obj.Target) == nil {
				return false
			}
		case quest.ObjectiveKillMonster, quest.ObjectiveSubmitPet:
			if s.gameDataManager.GetCreature(obj.Target) == nil {
				return false
			}
		default:
			return false
		}
	}
	return len(qp.Objectives) > 0
}
