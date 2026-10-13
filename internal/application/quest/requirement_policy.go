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
	PolicyUnconfigured   = "unconfigured"
	PolicyCurrency       = "unsupported_currency_condition"
	PolicyInvalid        = "invalid_catalog"
)

var reviewedConversationIDs = map[int]bool{
	1: true, 2: true, 3: true, 4: true, 6: true, 15: true, 23: true,
	24: true, 25: true, 26: true, 27: true, 28: true, 29: true,
	31: true, 32: true, 33: true, 34: true, 35: true, 36: true, 37: true, 38: true, 39: true, 40: true, 41: true, 42: true, 43: true, 44: true, 45: true, 46: true,
	54: true, 58: true, 161: true, 163: true, 165: true, 167: true, 169: true, 172: true, 173: true,
	382: true, 383: true, 384: true, 385: true, 386: true,
	1710: true, 1711: true, 1712: true, 1713: true,
	2563: true, 2564: true, 2565: true, 2566: true, 2567: true, 2568: true, 2571: true, 2578: true,
	3191: true, 3192: true, 3193: true, 3194: true, 3195: true, 3196: true, 3197: true, 3198: true, 3199: true, 3200: true, 3201: true, 5189: true,
}

func ClassifyQuestCompletion(tpl *models.QuestTemplate, requirements []*models.QuestRequireTemplate) (string, int) {
	if tpl == nil {
		return PolicyInvalid, 0
	}
	if tpl.Rn > 0 || tpl.Rn < 0 {
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
	if int(tpl.Type) == 1 && reviewedConversationIDs[int(tpl.ID)] && tpl.FinishNPC > 0 {
		return PolicyConversation, int(tpl.FinishNPC)
	}
	return PolicyUnconfigured, 0
}

func descriptionThreshold(name, prefix string) (int, bool) {
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(name, prefix))
	return n, err == nil && n > 0
}
