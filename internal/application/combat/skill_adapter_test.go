// Open-sourced by BaoLT

package combat

import (
	"testing"

	domaincombat "mcgame-server/internal/domain/combat"
	gamedatamodels "mcgame-server/internal/gamedata/models"
)

func TestResolveSkillEffectID_UsesObservedEncodedImpactEffect(t *testing.T) {
	tests := []struct {
		name         string
		gamedataType int
		want         int
	}{
		{name: "close", gamedataType: gamedataTypeClose, want: 2080130010008},
		{name: "remote", gamedataType: gamedataTypeRemote, want: 2080130010008},
		{name: "func hurt", gamedataType: gamedataTypeFuncHurt, want: 2080130010008},
		{name: "magic", gamedataType: gamedataTypeMagic, want: 2080130010008},
		{name: "magic bullet", gamedataType: gamedataTypeMagicBullet, want: 2080130010008},
		{name: "state add", gamedataType: gamedataTypeStateAdd, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveSkillEffectID(tc.gamedataType); got != tc.want {
				t.Fatalf("resolveSkillEffectID(%d) = %d, want %d", tc.gamedataType, got, tc.want)
			}
		})
	}
}

func TestBuildSkillInfoFromGamedata_PassesThroughEffectIDs(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:          12345,
		Name:        "Test",
		Type:        float64(gamedataTypeMagic),
		FrontEffID:  1111111111111,
		AttackEffID: 2222222222222,
		SkillEffID:  3333333333333,
		BulletID:    4444444444444,
	}
	info := buildSkillInfoFromGamedata(tpl)
	if info.Template.FrontEffID != 1111111111111 {
		t.Fatalf("FrontEffID = %d, want 1111111111111", info.Template.FrontEffID)
	}
	if info.Template.AttackEffID != 2222222222222 {
		t.Fatalf("AttackEffID = %d, want 2222222222222", info.Template.AttackEffID)
	}
	if info.Template.SkillEffID != 3333333333333 {
		t.Fatalf("SkillEffID = %d, want 3333333333333 (no fallback when DB has value)", info.Template.SkillEffID)
	}
	if info.Template.BulletID != 4444444444444 {
		t.Fatalf("BulletID = %d, want 4444444444444", info.Template.BulletID)
	}
}

func TestBuildSkillInfoFromGamedata_FallsBackWhenSkillEffZero(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:         12345,
		Name:       "Test",
		Type:       float64(gamedataTypeMagic),
		SkillEffID: 0,
	}
	info := buildSkillInfoFromGamedata(tpl)
	if info.Template.SkillEffID != 2080130010008 {
		t.Fatalf("SkillEffID = %d, want fallback 2080130010008", info.Template.SkillEffID)
	}
}

func TestBuildSkillInfoFromGamedata_NoFallbackForNonDamage(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:         12345,
		Type:       float64(gamedataTypeStateAdd),
		SkillEffID: 0,
	}
	info := buildSkillInfoFromGamedata(tpl)
	if info.Template.SkillEffID != 0 {
		t.Fatalf("SkillEffID = %d, want 0 (state-add has no default effect)", info.Template.SkillEffID)
	}
}

func TestBuildSkillInfoFromGamedataUsesTransformedAwakeningTemplateFields(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:          201,
		Name:        "Awakened Skill",
		Type:        float64(gamedataTypeMagic),
		TargetType:  8,
		AreaAttack:  4,
		TargetNum:   3,
		UseMP:       25,
		BuffID:      14,
		BuffRate:    75,
		BuffRound:   2,
		Description: "Gây 175% sát thương ma pháp",
	}

	info := buildSkillInfoFromGamedata(tpl)

	if info.Template.ID != 201 || info.Template.Multiplier != 1.75 || info.Template.MPCost != 25 {
		t.Fatalf("transformed template = %#v", info.Template)
	}
	if info.CombatTarget != domaincombat.TargetTeam || info.AreaType != domaincombat.AreaRandom || info.AreaSize != 3 {
		t.Fatalf("target mapping = target %v area %v size %d", info.CombatTarget, info.AreaType, info.AreaSize)
	}
	if info.BuffID != 14 || info.BuffRate != 0.75 || info.BuffDuration != 2 {
		t.Fatalf("buff mapping = id %d rate %v duration %d", info.BuffID, info.BuffRate, info.BuffDuration)
	}
}

func TestResolveCharacterSkillMultiplier(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        float64
	}{
		{name: "damage", description: "Gây ra tối thiểu 278% sát thương vật lý", want: 2.78},
		{name: "leading buff", description: "Tăng 30% tốc độ, gây 150% sát thương vật lý", want: 1.5},
		{name: "decimal comma", description: "Tạo thành 120,5% tấn công ma pháp", want: 1.205},
		{name: "buff only", description: "Tăng 200% tấn công vật lý", want: 0},
		{name: "ambiguous conditional", description: "Khi HP cao tạo thành 285% sát thương, khi HP thấp tạo thành 315% sát thương", want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveCharacterSkillMultiplier(tc.description); got != tc.want {
				t.Fatalf("resolveCharacterSkillMultiplier(%q) = %v, want %v", tc.description, got, tc.want)
			}
		})
	}
}

func TestBuildSkillInfoFromGamedata_UsesExactBachBienTemplatePlan(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:       6712,
		Name:     "Bách Biến Trùng Kích ",
		Kind:     1,
		Level:    1,
		Type:     float64(gamedataTypeClose),
		ExpSkill: 1,
		DexSkill: -1,
	}
	info := buildSkillInfoFromGamedata(tpl)
	if info.Plan == nil || len(info.Plan.Effects) != 1 {
		t.Fatalf("plan = %#v, want exact template plan", info.Plan)
	}
	effect := info.Plan.Effects[0]
	if effect.Kind != domaincombat.EffectPhysicalDamage || effect.Coefficient != 1.4 {
		t.Fatalf("damage effect = %#v, want physical x1.4", effect)
	}
}

func TestBuildSkillInfoFromGamedata_UsesExactTrungKichFollowUpPlan(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:       6750,
		Name:     "ㄨ Trùng Kích Truy Kích ㄨ",
		Kind:     1,
		Level:    3,
		Type:     float64(gamedataTypeClose),
		ExpSkill: 1,
		DexSkill: -1,
	}
	info := buildSkillInfoFromGamedata(tpl)
	if info.Plan == nil || info.Plan.Family != "Trùng Kích Truy Kích" || len(info.Plan.Effects) != 1 {
		t.Fatalf("plan = %#v, want exact Trùng Kích follow-up plan", info.Plan)
	}
	effect := info.Plan.Effects[0]
	if effect.Kind != domaincombat.EffectPhysicalDamage || effect.Coefficient != 4.0 {
		t.Fatalf("damage effect = %#v, want physical x4.0", effect)
	}
}

func TestBuildSkillInfoFromGamedata_UsesExactUMinhTemplatePlan(t *testing.T) {
	tpl := &gamedatamodels.SkillTemplate{
		ID:         7008,
		Name:       "ㄨU Minh Huyền Vũㄨ",
		Kind:       1,
		Level:      1,
		Type:       float64(gamedataTypeClose),
		TargetType: 4,
		TargetNum:  4,
		AreaAttack: 1,
		BuffID:     14,
		BuffRate:   60,
	}
	info := buildSkillInfoFromGamedata(tpl)
	if info.Plan == nil || info.Plan.Area != domaincombat.AreaHorizontalRows || info.Plan.TargetCount != 4 || len(info.Plan.Effects) != 2 {
		t.Fatalf("plan = %#v, want exact U Minh four-target horizontal-row plan", info.Plan)
	}
	damage := info.Plan.Effects[0]
	status := info.Plan.Effects[1]
	if damage.Coefficient != .25 || damage.CasterMaxHPRatio != .25 || status.Chance != .60 || status.BuffID != 14 {
		t.Fatalf("U Minh effects = %#v / %#v", damage, status)
	}
}
