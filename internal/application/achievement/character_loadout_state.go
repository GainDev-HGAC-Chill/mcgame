// Open-sourced by BaoLT

// Character-loadout state reader proves equipped quality, cosmetic, and magic-weapon milestones.
package achievement

import (
	"context"
	"fmt"
	"strings"

	appmagicweapon "mcgame-server/internal/application/magicweapon"
	domainitem "mcgame-server/internal/domain/item"
	domainmagicweapon "mcgame-server/internal/domain/magicweapon"
	"mcgame-server/internal/gamedata/models"
)

type CharacterLoadoutStateRepository interface {
	FindEquipped(ctx context.Context, characterID int64) ([]*domainitem.Item, error)
}

type CharacterLoadoutStateCatalog interface {
	GetEquipment(id int) *models.EquiptTemplateTemplate
}

type CharacterLoadoutStateReader struct {
	repo    CharacterLoadoutStateRepository
	catalog CharacterLoadoutStateCatalog
}

func NewCharacterLoadoutStateReader(repo CharacterLoadoutStateRepository, catalog CharacterLoadoutStateCatalog) *CharacterLoadoutStateReader {
	return &CharacterLoadoutStateReader{repo: repo, catalog: catalog}
}

func (r *CharacterLoadoutStateReader) Name() string {
	return "character-loadout"
}

func (r *CharacterLoadoutStateReader) RequirementIDs() []int64 {
	return []int64{
		23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 36, 37, 38, 342,
		360, 361, 362, 363, 364, 365, 366, 367, 368, 369, 370, 371, 372,
		377, 378, 499, 512, 515, 516, 569, 593, 710, 711,
		592, 594,
		636, 637, 638, 639, 640, 641, 642, 643,
	}
}

func (r *CharacterLoadoutStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("character-loadout state dependencies not configured")
	}
	items, err := r.repo.FindEquipped(ctx, characterID)
	if err != nil {
		return nil, err
	}
	standardColors := make(map[int]int)
	occupiedSlots := make(map[int]struct{})
	hasFashion := 0
	hasFlyer := 0
	hasMainMagicWeapon := 0
	subMagicWeapons := 0
	hasLevelTwentyMainMagicWeapon := 0
	hasStageSevenMaxMainMagicWeapon := 0
	hasStageEightMainMagicWeapon := 0
	hasOrangeSignature := 0
	hasWing := 0
	wingColors := make(map[int]int)
	exact := make(map[int64]int)
	standardSublimationElements := make(map[int]int)
	standardSublimationTwentyFive := 0
	standardSublimationFifty := 0
	for _, item := range items {
		if item == nil || item.ID <= 0 || item.CharacterID != characterID || item.ItemType != domainitem.ItemTypeEquipment || item.SlotType != domainitem.SlotTypeEquipped {
			continue
		}
		if item.SlotIndex < 0 || item.SlotIndex > 21 {
			return nil, fmt.Errorf("equipped item %d has invalid slot %d", item.ID, item.SlotIndex)
		}
		if _, duplicate := occupiedSlots[item.SlotIndex]; duplicate {
			return nil, fmt.Errorf("character %d repeats equipped slot %d", characterID, item.SlotIndex)
		}
		occupiedSlots[item.SlotIndex] = struct{}{}
		template := r.catalog.GetEquipment(item.TemplateID)
		if template == nil || template.ID != int64(item.TemplateID) {
			return nil, fmt.Errorf("equipped item %d has unknown template %d", item.ID, item.TemplateID)
		}
		position, positionOK := exactPetCatalogInt(template.Position)
		kind, kindOK := exactPetCatalogInt(template.Kind)
		if !positionOK || !kindOK {
			return nil, fmt.Errorf("equipped template %d has invalid topology", item.TemplateID)
		}
		switch {
		case position >= 1 && position <= 12:
			color, colorErr := petEquipmentStateColor(item)
			if colorErr != nil {
				return nil, fmt.Errorf("equipped item %d: %w", item.ID, colorErr)
			}
			standardColors[color]++
			sublimation, sublimationErr := decodeSublimationState(item.Properties["flag"])
			if sublimationErr != nil {
				return nil, fmt.Errorf("equipped item %d: %w", item.ID, sublimationErr)
			}
			if sublimation.Element > 0 {
				standardSublimationElements[sublimation.Element]++
			}
			if sublimation.Level >= 25 {
				standardSublimationTwentyFive++
			}
			if sublimation.Level >= 50 {
				standardSublimationFifty++
			}
			if color == 5 && strings.TrimSpace(itemPropertyString(item.Properties, "maker")) != "" {
				hasOrangeSignature = 1
			}
		case position == 14 && kind == 10:
			hasFlyer = 1
			markFlyerLoadoutRequirements(exact, template.ResCode)
		case position == 15 && kind == 8:
			hasMainMagicWeapon = 1
			if item.StarLevel >= 20 {
				hasLevelTwentyMainMagicWeapon = 1
			}
			stageSeven, stageEight, stageErr := mainMagicWeaponAchievementState(item, template)
			if stageErr != nil {
				return nil, fmt.Errorf("main magic weapon %d: %w", item.ID, stageErr)
			}
			if stageSeven {
				hasStageSevenMaxMainMagicWeapon = 1
			}
			if stageEight {
				hasStageEightMainMagicWeapon = 1
			}
		case position >= 16 && position <= 20 && kind == 8:
			subMagicWeapons++
		case position == 21 && kind == 11:
			hasFashion = 1
			markFashionLoadoutRequirements(exact, item.TemplateID)
		case position == 22 && kind == 13:
			color, colorErr := petEquipmentStateColor(item)
			if colorErr != nil {
				return nil, fmt.Errorf("equipped wing %d: %w", item.ID, colorErr)
			}
			hasWing = 1
			wingColors[color] = 1
			markWingLoadoutRequirements(exact, item.TemplateID, color)
		}
	}
	return []StateProof{
		stateProof(23, hasFashion, 1),
		stateProof(24, hasMainMagicWeapon, 1),
		stateProof(25, subMagicWeapons, 3),
		stateProof(26, hasFlyer, 1),
		stateProof(27, exact[27], 1),
		stateProof(28, exact[28], 1),
		stateProof(29, standardColors[2], 5),
		stateProof(30, standardColors[3], 5),
		stateProof(31, standardColors[4], 5),
		stateProof(32, standardColors[5], 5),
		stateProof(33, hasOrangeSignature, 1),
		stateProof(36, exact[36], 1),
		stateProof(37, exact[37], 1),
		stateProof(38, exact[38], 1),
		stateProof(342, exact[342], 1),
		stateProof(360, hasWing, 1),
		stateProof(361, wingColors[3], 1),
		stateProof(362, wingColors[4], 1),
		stateProof(363, wingColors[5], 1),
		stateProof(364, exact[364], 1),
		stateProof(365, exact[365], 1),
		stateProof(366, exact[366], 1),
		stateProof(367, exact[367], 1),
		stateProof(368, exact[368], 1),
		stateProof(369, exact[369], 1),
		stateProof(370, exact[370], 1),
		stateProof(371, exact[371], 1),
		stateProof(372, exact[372], 1),
		stateProof(377, exact[377], 1),
		stateProof(378, exact[378], 1),
		stateProof(499, exact[499], 1),
		stateProof(512, hasLevelTwentyMainMagicWeapon, 1),
		stateProof(515, exact[515], 1),
		stateProof(516, exact[516], 1),
		stateProof(569, exact[569], 1),
		stateProof(592, hasStageSevenMaxMainMagicWeapon, 1),
		stateProof(593, subMagicWeapons, 1),
		stateProof(594, hasStageEightMainMagicWeapon, 1),
		stateProof(710, exact[710], 1),
		stateProof(711, exact[711], 1),
		stateProof(636, standardSublimationElements[1], 12),
		stateProof(637, standardSublimationElements[2], 12),
		stateProof(638, standardSublimationElements[3], 12),
		stateProof(639, standardSublimationElements[4], 12),
		stateProof(640, standardSublimationElements[5], 12),
		stateProof(641, standardSublimationElements[6], 12),
		stateProof(642, standardSublimationTwentyFive, 12),
		stateProof(643, standardSublimationFifty, 12),
	}, nil
}

func mainMagicWeaponAchievementState(item *domainitem.Item, template *models.EquiptTemplateTemplate) (bool, bool, error) {
	if item == nil || template == nil || !domainmagicweapon.IsMainTemplate(template) {
		return false, false, nil
	}
	prop1, prop1Exists, prop1OK := exactLoadoutStateFloat(item.Properties, domainmagicweapon.PropKeyMainPropNum1)
	prop2, prop2Exists, prop2OK := exactLoadoutStateFloat(item.Properties, domainmagicweapon.PropKeyMainPropNum2)
	if !prop1Exists || !prop2Exists {
		return false, false, nil
	}
	if !prop1OK || !prop2OK || prop1 < 0 || prop2 < 0 || template.MainPropNum1 <= 0 || template.MainPropNum2 <= 0 {
		return false, false, fmt.Errorf("invalid main property state")
	}
	stage1 := appmagicweapon.ComputeStageVong(prop1, template.MainPropNum1)
	stage2 := appmagicweapon.ComputeStageVong(prop2, template.MainPropNum2)
	stageSeven := item.StarLevel >= domainmagicweapon.StageEightLevel && stage1 == 7 && stage2 == 7
	stageEight := stage1 >= 8 && stage2 >= 8
	return stageSeven, stageEight, nil
}

func exactLoadoutStateFloat(properties map[string]interface{}, key string) (float64, bool, bool) {
	value, exists := properties[key]
	if !exists || value == nil {
		return 0, false, true
	}
	switch typed := value.(type) {
	case int:
		return float64(typed), true, true
	case int32:
		return float64(typed), true, true
	case int64:
		return float64(typed), true, true
	case float32:
		return float64(typed), true, true
	case float64:
		return typed, true, true
	default:
		return 0, true, false
	}
}

func markFlyerLoadoutRequirements(progress map[int64]int, rawResCode float64) {
	resCode, ok := exactLoadoutCatalogInt64(rawResCode)
	if !ok {
		return
	}
	switch resCode {
	case 2070380000002:
		progress[342] = 1
	case 2070380000003:
		progress[27] = 1
	case 2070380000004:
		progress[28] = 1
	case 2070380000008:
		progress[377] = 1
	case 2070380000009:
		progress[378] = 1
	case 2070380000012:
		progress[499] = 1
	case 2070380000014, 2070380000015:
		progress[516] = 1
	case 2070380000016, 2070380000017:
		progress[515] = 1
	case 2070380000019:
		progress[569] = 1
	case 2070380000022:
		progress[710] = 1
	}
}

func markFashionLoadoutRequirements(progress map[int64]int, templateID int) {
	switch templateID {
	case 1852, 1853, 1975:
		progress[36] = 1
	case 1435, 1436, 1974:
		progress[37] = 1
	case 1636, 1637, 1972:
		progress[38] = 1
	case 2040, 2052, 2057:
		progress[711] = 1
	}
}

func markWingLoadoutRequirements(progress map[int64]int, templateID, color int) {
	baseRequirementID := int64(0)
	switch templateID {
	case 1676:
		baseRequirementID = 364
	case 1677:
		baseRequirementID = 365
	case 1678:
		baseRequirementID = 366
	}
	if baseRequirementID == 0 || color < 3 || color > 5 {
		return
	}
	progress[baseRequirementID+int64((color-3)*3)] = 1
}

func exactLoadoutCatalogInt64(value float64) (int64, bool) {
	result := int64(value)
	return result, float64(result) == value
}

func itemPropertyString(properties map[string]interface{}, key string) string {
	if value, ok := properties[key].(string); ok {
		return value
	}
	return ""
}
