// Catalog tests verify every checked-in Pet active skill template has an executable plan.
package combat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	domaincombat "mcgame-server/internal/domain/combat"
)

type catalogFixture struct {
	Records []map[string]string `json:"records"`
}

func TestPetActiveSkillCatalogCoversAuditedTemplates(t *testing.T) {
	items := readCatalogFixture(t, "TBL_ITEM_TEMPLATE.json")
	skills := readCatalogFixture(t, "TBL_SKILL.json")
	bookNames := make(map[string]bool)
	for _, item := range items.Records {
		if item["type"] == "506" && item["useType"] == "2" {
			bookNames[item["name"]] = true
		}
	}
	covered := 0
	families := make(map[string]bool)
	wantPlans := make(map[string]bool)
	for _, skill := range skills.Records {
		if skill["kind"] != "1" || !bookNames[skill["name"]] {
			continue
		}
		level, err := strconv.Atoi(skill["level"])
		if err != nil {
			t.Fatalf("invalid level for %q: %v", skill["name"], err)
		}
		if petActiveSkillPlan(skill["name"], level) == nil {
			t.Errorf("missing plan for %q level %d", skill["name"], level)
			continue
		}
		covered++
		families[skill["name"]] = true
		wantPlans[skill["name"]+"|"+skill["level"]] = true
	}
	for level := 1; level <= 3; level++ {
		wantPlans["Hổ Khiếu Sơn Lâm|"+strconv.Itoa(level)] = true
		wantPlans["Thợ Săn Sát Thủ|"+strconv.Itoa(level)] = true
		wantPlans["Tứ Phương Khiếp Đảm|"+strconv.Itoa(level)] = true
	}
	if covered != 144 {
		t.Fatalf("covered templates = %d, want 144", covered)
	}
	if len(families) != 52 {
		t.Fatalf("covered families = %d, want 52", len(families))
	}
	if len(petActiveSkillCatalog) != 55 {
		t.Fatalf("catalog families = %d, want 55", len(petActiveSkillCatalog))
	}
	registered := 0
	for family, levels := range petActiveSkillCatalog {
		for level := range levels {
			registered++
			key := family + "|" + strconv.Itoa(level)
			if !wantPlans[key] {
				t.Errorf("unexpected plan %q level %d", family, level)
			}
		}
	}
	if registered != 153 {
		t.Fatalf("registered plans = %d, want 153", registered)
	}
}

func TestHoKhieuSonLamUsesAllEnemyTargetsAndPerTargetDispel(t *testing.T) {
	damage := []float64{.8, 1, 1.2}
	dispel := []float64{.3, .4, .5}
	for level := 1; level <= 3; level++ {
		plan := petActiveSkillPlan("Hổ Khiếu Sơn Lâm", level)
		if plan == nil {
			t.Fatalf("missing level %d", level)
		}
		if plan.Target != domaincombat.TargetAllEnemy || plan.TargetCount != 10 {
			t.Fatalf("level %d target = %d/%d, want all enemies/10", level, plan.Target, plan.TargetCount)
		}
		if len(plan.Effects) != 2 || plan.Effects[0].Kind != domaincombat.EffectPhysicalDamage || plan.Effects[0].Coefficient != damage[level-1] {
			t.Fatalf("level %d damage effect = %#v", level, plan.Effects)
		}
		if plan.Effects[1].Kind != domaincombat.EffectRemoveOne || plan.Effects[1].Chance != dispel[level-1] {
			t.Fatalf("level %d dispel effect = %#v", level, plan.Effects[1])
		}
	}
}

func TestBachBienTrungKichExactTemplateDamagePlans(t *testing.T) {
	want := map[int]float64{
		6712: 1.4,
		6713: 1.7,
		6714: 2.0,
		6715: 1.6,
		6716: 1.9,
		6717: 2.5,
		6718: 1.9,
		6719: 2.2,
		6720: 3.0,
	}
	followUpCoefficient := map[int]float64{
		6712: 2.5,
		6713: 2.75,
		6714: 3.0,
		6715: 2.8,
		6716: 3.15,
		6717: 3.5,
		6718: 3.0,
		6719: 3.5,
		6720: 4.0,
	}
	followUpStart := map[int]int{6712: 6724, 6713: 6727, 6714: 6730, 6715: 6733, 6716: 6736, 6717: 6739, 6718: 6742, 6719: 6745, 6720: 6748}
	for id, coefficient := range want {
		plan := petActiveSkillPlanByTemplateID(id)
		if plan == nil || plan.Target != domaincombat.TargetEnemy || plan.TargetCount != 1 || len(plan.Effects) != 1 {
			t.Fatalf("template %d plan = %#v", id, plan)
		}
		effect := plan.Effects[0]
		if effect.Kind != domaincombat.EffectPhysicalDamage || effect.Coefficient != coefficient {
			t.Fatalf("template %d damage = %#v, want physical coefficient %.1f", id, effect, coefficient)
		}
		if plan.OnKill == nil || plan.OnKill.PassiveTargetCounts[6723] != 3 || plan.OnKill.Target != domaincombat.TargetRandomEnemy {
			t.Fatalf("template %d on-kill plan = %#v", id, plan.OnKill)
		}
		for count := 1; count <= 3; count++ {
			followUpID := followUpStart[id] + count - 1
			if got := plan.OnKill.TemplateIDs[count]; got != followUpID {
				t.Fatalf("template %d count %d follow-up ID = %d, want %d", id, count, got, followUpStart[id]+count-1)
			}
			followUp := petActiveSkillPlanByTemplateID(followUpID)
			if followUp == nil || followUp.Family != "Trùng Kích Truy Kích" || len(followUp.Effects) != 1 {
				t.Fatalf("template %d exact follow-up plan = %#v", followUpID, followUp)
			}
			if effect := followUp.Effects[0]; effect.Kind != domaincombat.EffectPhysicalDamage || effect.Coefficient != followUpCoefficient[id] {
				t.Fatalf("template %d follow-up effect = %#v", followUpID, effect)
			}
		}
	}
	if plan := petActiveSkillPlanByTemplateID(6721); plan != nil {
		t.Fatalf("passive template 6721 unexpectedly returned active damage plan: %#v", plan)
	}
}

func TestNhipDapTraiTimExactTemplateDamagePlans(t *testing.T) {
	want := map[int]float64{5363: 1.20, 5362: 1.50, 5361: 1.80}
	wantMax := map[int]int{5363: 5, 5362: 10, 5361: 15}
	for id, coefficient := range want {
		plan := petActiveSkillPlanByTemplateID(id)
		if plan == nil || !plan.ComboChain || plan.ComboChainMaxCasts != wantMax[id] || plan.ComboChainDamageStep != .20 || len(plan.Effects) != 1 {
			t.Fatalf("template %d plan = %#v", id, plan)
		}
		effect := plan.Effects[0]
		if effect.Kind != domaincombat.EffectMagicDamage || effect.Coefficient != coefficient {
			t.Fatalf("template %d effect = %#v, want magic coefficient %.2f", id, effect, coefficient)
		}
	}
}

func TestSpecialPetSupportSkillPlans(t *testing.T) {
	for level := 1; level <= 3; level++ {
		plan := petActiveSkillPlan("Thuốc Kích Thích", level)
		if plan == nil || plan.Target != domaincombat.TargetSelf || plan.CostsOnTarget || len(plan.Costs) != 1 {
			t.Fatalf("Thuốc Kích Thích level %d plan = %#v", level, plan)
		}
	}
	for level, amount := range []int{10, 15, 20} {
		plan := petActiveSkillPlan("Thợ Săn Sát Thủ", level+1)
		if plan == nil || plan.Target != domaincombat.TargetAllTeam || plan.TargetCount != 10 || len(plan.Effects) != 1 {
			t.Fatalf("Thợ Săn Sát Thủ level %d plan = %#v", level+1, plan)
		}
		effect := plan.Effects[0]
		if effect.Duration != 5 || effect.Polarity != domaincombat.PolarityBeneficial {
			t.Fatalf("Thợ Săn Sát Thủ level %d effect = %#v", level+1, effect)
		}
		if effect.BuffID != 80118+level {
			t.Fatalf("Thợ Săn Sát Thủ level %d buff ID = %d, want %d", level+1, effect.BuffID, 80118+level)
		}
		assertPlanModifier(t, effect, domaincombat.StatSpeed, float64(amount)/100, 0)
		assertPlanModifier(t, effect, domaincombat.StatAccuracy, float64(amount)/100, 0)
	}
	for level, amount := range []int{5, 8, 12} {
		plan := petActiveSkillPlan("Tứ Phương Khiếp Đảm", level+1)
		wantCount := []int{4, 6, 8}[level]
		wantChance := []float64{.1, .2, .3}[level]
		if plan == nil || plan.Target != domaincombat.TargetRandomEnemy || plan.TargetCount != wantCount || len(plan.Effects) != 2 {
			t.Fatalf("Tứ Phương Khiếp Đảm level %d plan = %#v", level+1, plan)
		}
		status := plan.Effects[0]
		if status.Duration != 1 || status.Polarity != domaincombat.PolarityHarmful {
			t.Fatalf("Tứ Phương Khiếp Đảm level %d status = %#v", level+1, status)
		}
		if status.BuffID != 80121+level {
			t.Fatalf("Tứ Phương Khiếp Đảm level %d buff ID = %d, want %d", level+1, status.BuffID, 80121+level)
		}
		for _, stat := range []domaincombat.CombatStat{domaincombat.StatAttack, domaincombat.StatMagicAttack, domaincombat.StatSpeed} {
			assertPlanModifier(t, status, stat, -float64(amount)/100, 0)
		}
		if plan.Effects[1].Kind != domaincombat.EffectRemoveAll || plan.Effects[1].Chance != wantChance {
			t.Fatalf("Tứ Phương Khiếp Đảm level %d dispel = %#v", level+1, plan.Effects[1])
		}
	}
}

func TestUMinhHuyenVuExactTemplatePlans(t *testing.T) {
	want := map[int]struct {
		level      int
		targets    int
		hpRatio    float64
		soulChance float64
	}{
		7002: {1, 4, .15, .50}, 7003: {2, 6, .20, .55}, 7004: {3, 8, .25, .60},
		7005: {1, 4, .20, .55}, 7006: {2, 6, .25, .60}, 7007: {3, 8, .27, .65},
		7008: {1, 4, .25, .60}, 7009: {2, 6, .27, .65}, 7010: {3, 8, .30, .70},
	}
	for id, expected := range want {
		plan := petActiveSkillPlanByTemplateID(id)
		if plan == nil || plan.Level != expected.level || plan.Target != domaincombat.TargetEnemy || plan.Area != domaincombat.AreaHorizontalRows || plan.TargetCount != expected.targets || len(plan.Effects) != 2 {
			t.Fatalf("template %d plan = %#v", id, plan)
		}
		damage := plan.Effects[0]
		if damage.Kind != domaincombat.EffectPhysicalDamage || damage.Coefficient != .25 || damage.CasterMaxHPRatio != expected.hpRatio {
			t.Fatalf("template %d damage = %#v", id, damage)
		}
		status := plan.Effects[1]
		if status.Kind != domaincombat.EffectStatus || status.Chance != expected.soulChance || status.Duration != 1 || status.BuffID != 14 || !status.Control.Stun {
			t.Fatalf("template %d Thất Hồn = %#v", id, status)
		}
	}
}

func assertPlanModifier(t *testing.T, effect domaincombat.SkillEffect, stat domaincombat.CombatStat, percent float64, flat int) {
	t.Helper()
	for _, modifier := range effect.Modifiers {
		if modifier.Stat == stat {
			if modifier.Percent != percent || modifier.Flat != flat {
				t.Fatalf("modifier %d = %#v, want percent %v flat %d", stat, modifier, percent, flat)
			}
			return
		}
	}
	t.Fatalf("missing modifier %d in %#v", stat, effect.Modifiers)
}

func TestPetActiveSkillCatalogRejectsUnknown(t *testing.T) {
	if plan := petActiveSkillPlan("not a Pet skill", 1); plan != nil {
		t.Fatalf("unknown skill returned plan: %#v", plan)
	}
}

func TestEveryPetDamagePlanHasPositiveDamageAndTargets(t *testing.T) {
	damagePlans := 0
	for family, levels := range petActiveSkillCatalog {
		for level, plan := range levels {
			hasDamage := false
			for _, effect := range plan.Effects {
				switch effect.Kind {
				case domaincombat.EffectPhysicalDamage, domaincombat.EffectMagicDamage, domaincombat.EffectFixedCurrentHPDamage, domaincombat.EffectFixedMaxHPDamage:
					hasDamage = true
					if effect.Coefficient <= 0 && effect.Fixed <= 0 {
						t.Errorf("%s level %d has non-positive direct damage: %#v", family, level, effect)
					}
				}
				if effect.Periodic != nil && effect.Periodic.Damage && effect.Periodic.Coefficient <= 0 && effect.Periodic.Fixed <= 0 {
					t.Errorf("%s level %d has non-positive periodic damage: %#v", family, level, effect.Periodic)
				}
			}
			if hasDamage {
				damagePlans++
				if plan.TargetCount <= 0 {
					t.Errorf("%s level %d has damage but no targets", family, level)
				}
			}
		}
	}
	if damagePlans == 0 {
		t.Fatal("Pet skill catalog has no direct damage plans")
	}
}

func TestPetActiveSkillProbabilityModifiersUsePercentagePoints(t *testing.T) {
	tests := []struct {
		family string
		level  int
		stat   domaincombat.CombatStat
		want   int
	}{
		{family: "Duy Trì Tốc Độ", level: 3, stat: domaincombat.StatDodge, want: 30},
		{family: "Tích Lực Sơ Cấp", level: 1, stat: domaincombat.StatCritical, want: 48},
		{family: "Tích Lực Trung Cấp", level: 3, stat: domaincombat.StatCritical, want: 67},
		{family: "Thuốc Kích Thích", level: 2, stat: domaincombat.StatCombo, want: 21},
		{family: "Thuốc Kích Thích", level: 2, stat: domaincombat.StatCritical, want: 21},
	}
	for _, tc := range tests {
		t.Run(tc.family+"/"+strconv.Itoa(tc.level)+"/"+strconv.Itoa(int(tc.stat)), func(t *testing.T) {
			plan := petActiveSkillPlan(tc.family, tc.level)
			if plan == nil || len(plan.Effects) != 1 {
				t.Fatalf("plan = %#v", plan)
			}
			for _, modifier := range plan.Effects[0].Modifiers {
				if modifier.Stat == tc.stat {
					if modifier.Flat != tc.want || modifier.Percent != 0 {
						t.Fatalf("modifier = %#v, want flat %d", modifier, tc.want)
					}
					return
				}
			}
			t.Fatalf("missing stat modifier %d", tc.stat)
		})
	}
}

func readCatalogFixture(t *testing.T, name string) catalogFixture {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var fixture catalogFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return fixture
}
