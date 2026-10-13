// Material grades use zero-based client colors, independently of equipment quality.
package item

import (
	"fmt"
	"math"

	domainitem "mcgame-server/internal/domain/item"
)

func (h *awardEffectHandler) awardItemColorCode(awardType, templateID, quality int, payload map[string]interface{}) (int, error) {
	value, explicit := payload["materialColorCode"]
	if explicit {
		if awardType != 29 || h.gameData == nil {
			return 0, fmt.Errorf("material color requires an item-template award")
		}
		tpl := h.gameData.GetItem(templateID)
		if tpl == nil || int(tpl.Kind) != 6 {
			return 0, fmt.Errorf("material color requires a material template: %d", templateID)
		}
		var color float64
		switch v := value.(type) {
		case float64:
			color = v
		case int:
			color = float64(v)
		default:
			return 0, fmt.Errorf("invalid material color type")
		}
		if math.IsNaN(color) || math.IsInf(color, 0) || math.Trunc(color) != color || color < 1 || color > 4 {
			return 0, fmt.Errorf("explicit material color must be an integer from 1 to 4")
		}
		return int(color), nil
	}
	if quality > 0 {
		return domainitem.EquipmentColorCodeFromQuality(quality), nil
	}
	return 0, nil
}
