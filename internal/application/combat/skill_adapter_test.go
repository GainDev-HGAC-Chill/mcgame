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
