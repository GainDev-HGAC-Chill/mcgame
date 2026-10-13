package pet

import (
	"reflect"
	"strconv"
	"testing"
)

func TestPPVEDataHasAllEmptyCarvePartsWithoutProgression(t *testing.T) {
	service := &PPVEService{}
	for _, state := range []*PPVEState{defaultPPVEState(), {}} {
		data := service.PPVEData(state)
		parts := data["p"].(map[string]interface{})
		if len(parts) != 8 || data["mlv"] != 0 {
			t.Fatalf("empty panel DTO = %#v", data)
		}
		for index := 0; index < 8; index++ {
			part, ok := parts["s"+strconv.Itoa(index)].(map[string]interface{})
			if !ok || part == nil || len(part) != 0 {
				t.Fatalf("part %d = %#v", index, part)
			}
		}
		if len(state.P) != 0 || state.MasterLevel != 0 || state.KP != 0 || state.PPVEFloor != 0 {
			t.Fatalf("wire normalization changed progression: %#v", state)
		}
	}
}

func TestPPVEDataPreservesSavedRunesAndDoesNotPersistNormalization(t *testing.T) {
	state := &PPVEState{
		P: map[string]interface{}{
			"s0": map[string]interface{}{"0": float64(1120)},
			"s1": nil,
			"s2": "invalid",
		},
		MasterLevel: 2,
		PPVEConfig:  map[string]interface{}{"pets": "saved"},
	}
	before := encodePPVEState(state)
	wire := (&PPVEService{}).PPVEData(state)
	parts := wire["p"].(map[string]interface{})
	if !reflect.DeepEqual(parts["s0"], state.P["s0"]) || wire["mlv"] != 2 || !reflect.DeepEqual(wire["ppveConfig"], state.PPVEConfig) {
		t.Fatalf("saved values lost: %#v", wire)
	}
	for _, key := range []string{"s1", "s2"} {
		if part, ok := parts[key].(map[string]interface{}); !ok || part == nil || len(part) != 0 {
			t.Fatalf("invalid part %s not normalized: %#v", key, parts[key])
		}
	}
	if !reflect.DeepEqual(before, encodePPVEState(state)) || len(state.P) != 3 {
		t.Fatalf("persisted shape changed: %#v", state.P)
	}
	delete(parts, "s0")
	if state.P["s0"] == nil {
		t.Fatal("wire part containers alias the persisted outer map")
	}
}
