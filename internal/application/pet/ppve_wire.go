// Pet PVE requires eight carve-part containers even when no runes are unlocked.
// Wire normalization does not alter persisted state or award progression.
package pet

import "strconv"

func ppvePartsToWire(parts map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(parts)+8)
	for key, value := range parts {
		out[key] = value
	}
	for index := 0; index < 8; index++ {
		key := "s" + strconv.Itoa(index)
		if value, ok := out[key].(map[string]interface{}); !ok || value == nil {
			out[key] = map[string]interface{}{}
		}
	}
	return out
}
