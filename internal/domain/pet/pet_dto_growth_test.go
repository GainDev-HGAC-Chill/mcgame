package pet

import "testing"

func TestToDTOSeparatesBaseAddedAndEffectiveGrowth(t *testing.T) {
	pet := &Pet{ID: 1, GrowRate: 1.2, GrowRateAdd: 0.25, Property: map[string]interface{}{}}
	wrapped := pet.ToDTO()
	dto, ok := wrapped["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data type = %T", wrapped["data"])
	}
	if dto["growRate"] != 1.2 || dto["growRateAdd"] != 0.25 {
		t.Fatalf("root growth fields = %v/%v", dto["growRate"], dto["growRateAdd"])
	}
	property, ok := dto["property"].(map[string]interface{})
	if !ok {
		t.Fatalf("property type = %T", dto["property"])
	}
	if property["growRate"] != 1.2 || property["growRateAdd"] != 0.25 {
		t.Fatalf("property growth fields = %v/%v", property["growRate"], property["growRateAdd"])
	}
}

func TestToDTOPreservesPetSettingsAtWrapperRoot(t *testing.T) {
	pet := &Pet{ID: 1, Property: map[string]interface{}{"bs8": 3319, "bt8": 52}}
	dto := pet.ToDTO()
	pi, ok := dto["pi"].(map[string]interface{})
	if !ok || pi["bs8"] != 3319 || pi["bt8"] != 52 {
		t.Fatalf("wrapper pi = %#v", dto["pi"])
	}
	if dto["soulInfo"] == nil {
		t.Fatal("wrapper soulInfo missing")
	}
}
