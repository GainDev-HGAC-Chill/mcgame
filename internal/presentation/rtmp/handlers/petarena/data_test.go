package petarena

import (
	"testing"

	apppetarena "mcgame-server/internal/application/petarena"
)

func TestPetFightConfigContextUsesExistingXiulianshiArgument(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
		key  string
		flag bool
	}{
		{name: "farm", args: []interface{}{map[string]interface{}{}, float64(1), false}, key: apppetarena.ConfigContextFarm},
		{name: "hmtxls bool", args: []interface{}{map[string]interface{}{}, float64(1), true}, key: apppetarena.ConfigContextHMTXLS, flag: true},
		{name: "hmtxls amf number", args: []interface{}{map[string]interface{}{}, float64(1), float64(1)}, key: apppetarena.ConfigContextHMTXLS, flag: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key, flag := petFightConfigContext(test.args)
			if key != test.key || flag != test.flag {
				t.Fatalf("context = %q,%v, want %q,%v", key, flag, test.key, test.flag)
			}
		})
	}
}

func TestNormalizeStringMapAcceptsAMFMap(t *testing.T) {
	result, ok := normalizeStringMap(map[interface{}]interface{}{float64(5): map[string]interface{}{"pos": 5}})
	if !ok || result["5"] == nil {
		t.Fatalf("normalized map = %v, %v", result, ok)
	}
}
