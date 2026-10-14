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

func TestToDTOIncludesTooltipAttributesAndSkills(t *testing.T) {
	pet := &Pet{
		ID: 1,
		CreatureData: map[string]interface{}{
			"attStrength": "11", "attAgility": "12", "attStamina": "13", "attIntelligence": "14", "attEnergy": "15",
		},
		Property: map[string]interface{}{
			"finalStrength": 21, "finalAgility": 22, "finalStamina": 23, "finalIntelligence": 24, "finalEnergy": 25,
			"skill1": 7001, "skill2": 7002,
		},
	}
	dto := pet.ToDTO()["data"].(map[string]interface{})
	for key, want := range map[string]int{
		"attStrength": 21, "attAgility": 22, "attStamina": 23, "attIntelligence": 24, "attEnergy": 25,
		"skill1": 7001, "skill2": 7002,
	} {
		if got := dto[key]; got != want {
			t.Fatalf("%s = %#v, want %d", key, got, want)
		}
	}
}
