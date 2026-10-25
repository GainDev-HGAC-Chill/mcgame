// Open-sourced by BaoLT

// Sublimation state helpers decode the persisted equipment flag contract.
package achievement

import (
	"encoding/json"
	"fmt"
)

type sublimationState struct {
	Level   int
	Element int
}

func decodeSublimationState(raw interface{}) (sublimationState, error) {
	if raw == nil || raw == "" {
		return sublimationState{}, nil
	}
	values, ok := raw.(map[string]interface{})
	if !ok {
		text, stringOK := raw.(string)
		if !stringOK {
			return sublimationState{}, fmt.Errorf("invalid sublimation flag type")
		}
		if err := json.Unmarshal([]byte(text), &values); err != nil {
			return sublimationState{}, fmt.Errorf("decode sublimation flag: %w", err)
		}
	}
	level, err := optionalSublimationInt(values, "sublimeId")
	if err != nil || level < 0 || level > 50 {
		return sublimationState{}, fmt.Errorf("invalid sublimeId")
	}
	element, err := optionalSublimationInt(values, "sublimeElement")
	if err != nil || element < 0 || element > 6 {
		return sublimationState{}, fmt.Errorf("invalid sublimeElement")
	}
	if level == 0 && element != 0 {
		return sublimationState{}, fmt.Errorf("sublimation element without level")
	}
	return sublimationState{Level: level, Element: element}, nil
}

func optionalSublimationInt(values map[string]interface{}, key string) (int, error) {
	value, exists := values[key]
	if !exists || value == nil || value == "" {
		return 0, nil
	}
	parsed, ok := exactPetEquipmentStateInt(value)
	if !ok {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return parsed, nil
}
