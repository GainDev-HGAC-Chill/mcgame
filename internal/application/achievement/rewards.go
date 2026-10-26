// Open-sourced by BaoLT

// Achievement reward registry contains only website-and-catalog verified rewards.
package achievement

import (
	"fmt"

	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/gamedata/models"
)

type RewardKind int

const (
	RewardKindItem RewardKind = iota + 1
	RewardKindTitle
	RewardKindMedal
)

type RewardDefinition struct {
	Kind       RewardKind
	TemplateID int
	Quantity   int
}

type RewardCatalog interface {
	GetAchievement(id int) *models.AchievementTemplate
	GetBuff(id int) *models.BuffTemplate
	GetItem(id int) *models.ItemTemplateTemplate
	GetTitle(id int) *models.TitleTemplate
	GetMedal(id int) *models.MedalTemplate
}

var achievementRewards = map[int64]RewardDefinition{
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
	6066: {Kind: RewardKindTitle, TemplateID: 274, Quantity: 1},
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

func RewardForAchievement(achievementID int64) (RewardDefinition, bool) {
	reward, ok := achievementRewards[achievementID]
	return reward, ok
}

func ValidateRewardReceipt(achievementID int64, receipt domainachievement.RewardReceipt) error {
	reward, ok := RewardForAchievement(achievementID)
	if !ok {
		return ErrAchievementRewardUnknown
	}
	switch reward.Kind {
	case RewardKindItem:
		if receipt.ItemID <= 0 {
			return fmt.Errorf("invalid achievement item receipt")
		}
	case RewardKindTitle:
		if receipt.TitleID != reward.TemplateID {
			return fmt.Errorf("invalid achievement title receipt")
		}
	case RewardKindMedal:
		if receipt.MedalSlot <= 0 || receipt.MedalCount < reward.Quantity {
			return fmt.Errorf("invalid achievement medal receipt")
		}
	default:
		return fmt.Errorf("unsupported achievement reward kind")
	}
	return nil
}

func ValidateRewardCatalog(catalog RewardCatalog) error {
	if catalog == nil {
		return fmt.Errorf("achievement reward catalog is not configured")
	}
	for achievementID, reward := range achievementRewards {
		template := catalog.GetAchievement(int(achievementID))
		if template == nil || int(template.IsAward) <= 0 {
			return fmt.Errorf("achievement reward %d does not reference a reward-bearing achievement", achievementID)
		}
		switch reward.Kind {
		case RewardKindItem:
			if catalog.GetItem(reward.TemplateID) == nil {
				return fmt.Errorf("achievement %d references missing item %d", achievementID, reward.TemplateID)
			}
		case RewardKindTitle:
			if catalog.GetTitle(reward.TemplateID) == nil {
				return fmt.Errorf("achievement %d references missing title %d", achievementID, reward.TemplateID)
			}
		case RewardKindMedal:
			if catalog.GetMedal(reward.TemplateID) == nil {
				return fmt.Errorf("achievement %d references missing medal %d", achievementID, reward.TemplateID)
			}
		default:
			return fmt.Errorf("achievement %d has unsupported reward kind %d", achievementID, reward.Kind)
		}
		if reward.Quantity <= 0 {
			return fmt.Errorf("achievement %d has invalid reward quantity", achievementID)
		}
	}
	title := catalog.GetTitle(274)
	if title == nil || int(title.B) != 2793 {
		return fmt.Errorf("achievement 6066 title authority is invalid")
	}
	buff := catalog.GetBuff(2793)
	if buff == nil || buff.Name != "Giáo Sư Biết Tuốt" {
		return fmt.Errorf("achievement 6066 linked title buff authority is invalid")
	}
	return nil
}
