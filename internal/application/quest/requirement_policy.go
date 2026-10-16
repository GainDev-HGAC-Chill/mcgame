// Completion policy is derived from catalog requirements or explicitly reviewed local descriptions.
// Unknown empty definitions and unimplemented currency conditions fail closed, regardless of stored objectives.
package quest

import (
	"strconv"
	"strings"

	"mcgame-server/internal/gamedata/models"
)

const (
	PolicyCatalog        = "catalog"
	PolicyConversation   = "reviewed_conversation"
	PolicyCharacterLevel = "character_level"
	PolicyPetLevel       = "pet_level"
	PolicyPetCount       = "pet_count"
	PolicyFriendship     = "friendship"
	PolicyClassRank      = "class_rank"
	PolicyUnconfigured   = "unconfigured"
	PolicyCurrency       = "unsupported_currency_condition"
	PolicyInvalid        = "invalid_catalog"
)

func ClassifyQuestCompletion(tpl *models.QuestTemplate, requirements []*models.QuestRequireTemplate) (string, int) {
	if tpl == nil {
		return PolicyInvalid, 0
	}
	if tpl.Rn != 0 {
		return PolicyCurrency, 0
	}
	if len(requirements) > 0 {
		for _, row := range requirements {
			if row == nil || row.ItemID <= 0 || row.ItemID != float64(int(row.ItemID)) || row.Num <= 0 || row.Num != float64(int(row.Num)) {
				return PolicyInvalid, 0
			}
			if (row.Kind == 1 && (row.Type == 19 || row.Type == 29)) || ((row.Kind == 2 || row.Kind == 3) && row.Type == 12) {
				continue
			}
			return PolicyInvalid, 0
		}
		return PolicyCatalog, 0
	}
	if tpl.Rt != 0 {
		return PolicyCurrency, 0
	}
	if spec, ok := reviewedClassRankQuests[int(tpl.ID)]; ok && classRankQuestFingerprintMatches(tpl, spec) {
		return PolicyClassRank, spec.ClassRank
	}
	if int(tpl.Type) == 10 {
		id := int(tpl.ID)
		if id >= 2695 && id <= 2733 && tpl.Name == "Kết giao "+strconv.Itoa(id-2694)+" người bạn" {
			return PolicyFriendship, id - 2694
		}
		if id >= 2590 && id <= 2618 {
			if n, ok := descriptionThreshold(tpl.Name, "Cấp Độ Lên Đến "); ok {
				return PolicyCharacterLevel, n
			}
		}
		if id >= 2629 && id <= 2657 {
			if n, ok := descriptionThreshold(tpl.Name, "Cấp độ pet tăng đến "); ok {
				return PolicyPetLevel, n
			}
		}
		if id >= 2663 && id <= 2685 {
			if n, ok := descriptionThreshold(tpl.Name, "Tăng số lượng pet lên "); ok {
				return PolicyPetCount, n
			}
		}
	}
	if spec, ok := reviewedConversationQuests[int(tpl.ID)]; ok && conversationFingerprintMatches(tpl, spec) {
		return PolicyConversation, spec.FinishNPC
	}
	return PolicyUnconfigured, 0
}

func conversationFingerprintMatches(tpl *models.QuestTemplate, spec conversationQuestSpec) bool {
	return tpl != nil &&
		tpl.Type == float64(spec.QuestType) &&
		tpl.StartNPC == float64(spec.StartNPC) &&
		tpl.FinishNPC == float64(spec.FinishNPC) &&
		spec.FinishNPC > 0 &&
		tpl.Name == spec.Name &&
		strings.TrimSpace(tpl.StartText) != "" &&
		strings.TrimSpace(tpl.CompleteText) != ""
}

func descriptionThreshold(name, prefix string) (int, bool) {
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(name, prefix))
	return n, err == nil && n > 0
}
