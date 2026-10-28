package guild

import "testing"

func TestRankSettingAllowsExactPermission(t *testing.T) {
	setting := RankSetting{Rank: RankOfficer, Name: "Trưởng lão", CanAdd: true, CanSlot: true}
	if !setting.Allows(PermissionAdd) || !setting.Allows(PermissionSlot) {
		t.Fatal("enabled permissions were denied")
	}
	for _, permission := range []Permission{PermissionQuest, PermissionInfo, PermissionDel, PermissionDuty, Permission("unknown")} {
		if setting.Allows(permission) {
			t.Fatalf("disabled permission %q was allowed", permission)
		}
	}
	if (RankSetting{Rank: RankLeader}).Allows(Permission("unknown")) {
		t.Fatal("unknown permission must not be granted to leader")
	}
}

func TestValidateRankSettingsRejectsIncompleteAndMutableLeader(t *testing.T) {
	settings := []RankSetting{{Rank: RankLeader, Name: "Bang chủ"}}
	if ValidateRankSettings(settings) == nil {
		t.Fatal("incomplete settings must fail")
	}
	settings = make([]RankSetting, 0, RankMember)
	for rank := RankLeader; rank <= RankMember; rank++ {
		settings = append(settings, RankSetting{Rank: rank, Name: "rank"})
	}
	if ValidateRankSettings(settings) == nil {
		t.Fatal("disabled leader permissions must fail")
	}
	settings[0].CanAdd = true
	settings[0].CanQuest = true
	settings[0].CanSlot = true
	settings[0].CanInfo = true
	settings[0].CanDel = true
	settings[0].CanDuty = true
	if err := ValidateRankSettings(settings); err != nil {
		t.Fatal(err)
	}
}

func TestRankSettingsFromDTOSupportsLegacyIntegerFlags(t *testing.T) {
	values := make([]map[string]interface{}, 0, RankMember)
	for rank := RankLeader; rank <= RankMember; rank++ {
		values = append(values, map[string]interface{}{
			"rank": float64(rank), "name": "rank", "canAdd": float64(1), "canQuest": float64(1),
			"canSlot": float64(1), "canInfo": float64(1), "canDel": float64(1), "canDuty": float64(1),
		})
	}
	settings, err := RankSettingsFromDTO(values)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) != RankMember || !settings[RankMember-1].CanDuty {
		t.Fatalf("settings = %#v", settings)
	}
}
