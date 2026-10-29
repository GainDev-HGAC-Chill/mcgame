// Ordinary gathering registries are explicit catalog allow-lists.
package gathering

import (
	"fmt"

	domaingathering "mcgame-server/internal/domain/gathering"
	"mcgame-server/internal/gamedata/models"
)

const (
	fishItemCatalogType = 610
	herbItemCatalogType = 612
)

type LevelRewards struct {
	Level       int
	TemplateIDs [2]int
}

var ordinaryFishRewards = [...]LevelRewards{
	{Level: 1, TemplateIDs: [2]int{2361, 2362}},
	{Level: 2, TemplateIDs: [2]int{2363, 2364}},
	{Level: 3, TemplateIDs: [2]int{2365, 2366}},
	{Level: 4, TemplateIDs: [2]int{2367, 2368}},
	{Level: 5, TemplateIDs: [2]int{2369, 2370}},
	{Level: 6, TemplateIDs: [2]int{2589, 2588}},
	{Level: 7, TemplateIDs: [2]int{2591, 2592}},
	{Level: 8, TemplateIDs: [2]int{2594, 2595}},
	{Level: 9, TemplateIDs: [2]int{2597, 2599}},
	{Level: 10, TemplateIDs: [2]int{2600, 2602}},
}

var ordinaryHerbRewards = [...]LevelRewards{
	{Level: 1, TemplateIDs: [2]int{2491, 2492}},
	{Level: 2, TemplateIDs: [2]int{2493, 2494}},
	{Level: 3, TemplateIDs: [2]int{2495, 2496}},
	{Level: 4, TemplateIDs: [2]int{2497, 2498}},
	{Level: 5, TemplateIDs: [2]int{2499, 2500}},
	{Level: 6, TemplateIDs: [2]int{2604, 2603}},
	{Level: 7, TemplateIDs: [2]int{2606, 2605}},
	{Level: 8, TemplateIDs: [2]int{2607, 2608}},
	{Level: 9, TemplateIDs: [2]int{2609, 2610}},
	{Level: 10, TemplateIDs: [2]int{2611, 2612}},
}

var ordinaryFishingNodeLevels = map[int]int{
	1189: 1, 1190: 1, 1191: 1, 1192: 1, 1193: 1,
	1194: 1, 1195: 1, 1196: 1, 1197: 1, 1198: 1,
	1199: 2, 1202: 2, 1203: 2, 1204: 2,
	1205: 3, 1206: 3, 1207: 3,
	1208: 4, 1209: 4, 1210: 4, 1211: 4,
	1212: 5, 1213: 5, 1214: 5, 1215: 5,
	1216: 6, 1217: 6, 1218: 6,
	1596: 7, 1597: 7, 1598: 7,
}

var herbTemplateLevels = map[int]int{
	1241: 1, 1242: 2, 1243: 3, 1244: 4, 1245: 5,
	1246: 6, 1247: 7, 1248: 8, 1249: 9, 1250: 10,
}

var excludedSpecialFish = map[int]struct{}{
	2590: {},
	2596: {},
	2598: {},
	2601: {},
}

type ItemCatalog interface {
	GetItem(id int) *models.ItemTemplateTemplate
}

func OrdinaryRewards(kind domaingathering.ResourceKind, level int) ([2]int, bool) {
	registry := ordinaryRewardsForKind(kind)
	if level <= 0 || level > len(registry) {
		return [2]int{}, false
	}
	entry := registry[level-1]
	if entry.Level != level {
		return [2]int{}, false
	}
	return entry.TemplateIDs, true
}

func IsOrdinaryReward(kind domaingathering.ResourceKind, level int, templateID int) bool {
	rewards, ok := OrdinaryRewards(kind, level)
	return ok && (rewards[0] == templateID || rewards[1] == templateID)
}

func OrdinaryFishingNodeLevel(npcID int) (int, bool) {
	level, ok := ordinaryFishingNodeLevels[npcID]
	return level, ok
}

func HerbTemplateLevel(npcID int) (int, bool) {
	level, ok := herbTemplateLevels[npcID]
	return level, ok
}

func IsExcludedSpecialFish(templateID int) bool {
	_, ok := excludedSpecialFish[templateID]
	return ok
}

func MasteryGain(skillLevel int, resourceLevel int) int {
	switch skillLevel - resourceLevel {
	case 0:
		return 2
	case 1:
		return 1
	default:
		return 0
	}
}

func ValidateOrdinaryCatalog(catalog ItemCatalog) error {
	if catalog == nil {
		return fmt.Errorf("gathering item catalog is unavailable")
	}
	if err := validateRewardCatalog(catalog, domaingathering.ResourceKindFishing, fishItemCatalogType); err != nil {
		return err
	}
	if err := validateRewardCatalog(catalog, domaingathering.ResourceKindHerb, herbItemCatalogType); err != nil {
		return err
	}
	for templateID := range excludedSpecialFish {
		for _, entry := range ordinaryFishRewards {
			if entry.TemplateIDs[0] == templateID || entry.TemplateIDs[1] == templateID {
				return fmt.Errorf("special fish %d appears in ordinary registry", templateID)
			}
		}
	}
	return nil
}

func ordinaryRewardsForKind(kind domaingathering.ResourceKind) []LevelRewards {
	switch kind {
	case domaingathering.ResourceKindFishing:
		return ordinaryFishRewards[:]
	case domaingathering.ResourceKindHerb:
		return ordinaryHerbRewards[:]
	default:
		return nil
	}
}

func validateRewardCatalog(catalog ItemCatalog, kind domaingathering.ResourceKind, expectedType int) error {
	for _, entry := range ordinaryRewardsForKind(kind) {
		for _, templateID := range entry.TemplateIDs {
			template := catalog.GetItem(templateID)
			if template == nil {
				return fmt.Errorf("gathering reward %d is missing", templateID)
			}
			if int(template.Type) != expectedType || int(template.ItemLevel) != entry.Level {
				return fmt.Errorf("gathering reward %d has type %d level %d, expected type %d level %d", templateID, int(template.Type), int(template.ItemLevel), expectedType, entry.Level)
			}
		}
	}
	return nil
}
