// Forum skill tests pin exact seed-backed plans and prevent legacy fallback.
package combat

import (
	"testing"

	domaincombat "mcgame-server/internal/domain/combat"
)

func TestForumExactDamagePlans(t *testing.T) {
	tests := map[int]struct {
		family      string
		kind        domaincombat.EffectKind
		coefficient float64
		area        domaincombat.AreaType
		targets     int
		rest        int
	}{
		2961: {"Cao Tần Long Hống Trung Cấp", domaincombat.EffectMagicDamage, .95, domaincombat.AreaNone, 1, 0},
		2962: {"Cao Tần Long Hống Trung Cấp", domaincombat.EffectMagicDamage, 1.10, domaincombat.AreaNone, 1, 0},
		2963: {"Cao Tần Long Hống Trung Cấp", domaincombat.EffectMagicDamage, 1.25, domaincombat.AreaNone, 1, 0},
		2666: {"Đột Thích Xung Phong", domaincombat.EffectPhysicalDamage, .80, domaincombat.AreaNone, 1, 0},
		2735: {"Đột Thích Xung Phong", domaincombat.EffectPhysicalDamage, .90, domaincombat.AreaNone, 1, 0},
		2736: {"Đột Thích Xung Phong", domaincombat.EffectPhysicalDamage, 1.00, domaincombat.AreaNone, 1, 0},
		2692: {"Lạc Thạch Vẫn Tinh", domaincombat.EffectMagicDamage, .30, domaincombat.AreaHorizontal, 3, 0},
		2777: {"Lạc Thạch Vẫn Tinh", domaincombat.EffectMagicDamage, .40, domaincombat.AreaHorizontal, 3, 0},
		2784: {"Lạc Thạch Vẫn Tinh", domaincombat.EffectMagicDamage, .50, domaincombat.AreaHorizontal, 3, 0},
		3542: {"Táng Thần Quần Thể", domaincombat.EffectPhysicalDamage, .90, domaincombat.AreaCross, 4, 1},
		3544: {"Táng Thần Quần Thể", domaincombat.EffectPhysicalDamage, 1.08, domaincombat.AreaCross, 4, 1},
		3546: {"Táng Thần Quần Thể", domaincombat.EffectPhysicalDamage, 1.28, domaincombat.AreaCross, 4, 1},
		2725: {"Uy Nhiếp Ý Chí", domaincombat.EffectMagicDamage, .30, domaincombat.AreaHorizontal, 3, 0},
		2690: {"Uy Nhiếp Ý Chí", domaincombat.EffectMagicDamage, .40, domaincombat.AreaHorizontal, 3, 0},
		2727: {"Uy Nhiếp Ý Chí", domaincombat.EffectMagicDamage, .50, domaincombat.AreaHorizontal, 3, 0},
	}
	for id, expected := range tests {
		plan := petActiveSkillPlanByTemplateID(id)
		if plan == nil || plan.Family != expected.family || plan.Area != expected.area || plan.TargetCount != expected.targets || plan.RestTurns != expected.rest || len(plan.Effects) == 0 {
			t.Fatalf("template %d plan = %#v", id, plan)
		}
		effect := plan.Effects[0]
		if effect.Kind != expected.kind || effect.Coefficient != expected.coefficient {
			t.Fatalf("template %d damage = %#v", id, effect)
		}
	}
}

func TestForumExactStatusPlans(t *testing.T) {
	tests := map[int]struct {
		family   string
		chance   float64
		duration int
		buffID   int
		targets  int
		area     domaincombat.AreaType
	}{
		2845: {"Kích Trạng Loạn Tâm", .05, 2, 187, 1, domaincombat.AreaNone},
		2839: {"Kích Trạng Loạn Tâm", .10, 2, 187, 1, domaincombat.AreaNone},
		2846: {"Kích Trạng Loạn Tâm", .15, 2, 187, 1, domaincombat.AreaNone},
		4739: {"Mê Tâm", .60, 2, 187, 2, domaincombat.AreaNone},
		4741: {"Mê Tâm", .65, 2, 187, 3, domaincombat.AreaNone},
		4742: {"Mê Tâm", .70, 3, 187, 4, domaincombat.AreaNone},
		2838: {"Phản Ứng Trì Độn", .05, 2, 497, 1, domaincombat.AreaNone},
		2843: {"Phản Ứng Trì Độn", .10, 2, 498, 1, domaincombat.AreaNone},
		2844: {"Phản Ứng Trì Độn", .15, 2, 499, 1, domaincombat.AreaNone},
		3540: {"Trào Phúng Quần Thể", .66, 1, 893, 2, domaincombat.AreaVertical},
		3566: {"Trào Phúng Quần Thể", .66, 2, 893, 2, domaincombat.AreaVertical},
		3567: {"Trào Phúng Quần Thể", .66, 2, 893, 4, domaincombat.AreaCross},
		2615: {"Thạch Hóa Công Kích", .05, 2, 299, 1, domaincombat.AreaNone},
	}
	for id, expected := range tests {
		plan := petActiveSkillPlanByTemplateID(id)
		if plan == nil || plan.Family != expected.family || plan.Area != expected.area || plan.TargetCount != expected.targets {
			t.Fatalf("template %d plan = %#v", id, plan)
		}
		effect := plan.Effects[len(plan.Effects)-1]
		if effect.Kind != domaincombat.EffectStatus || effect.Chance != expected.chance || effect.Duration != expected.duration || effect.BuffID != expected.buffID {
			t.Fatalf("template %d status = %#v", id, effect)
		}
	}
	if effect := petActiveSkillPlanByTemplateID(2615).Effects[0]; !effect.Control.Stone {
		t.Fatalf("stone effect = %#v", effect)
	}
}

func TestForumDamageDebuffPlans(t *testing.T) {
	for _, id := range []int{2961, 2962, 2963, 2666, 2735, 2736} {
		plan := petActiveSkillPlanByTemplateID(id)
		if plan == nil || len(plan.Effects) != 2 || plan.Effects[1].Kind != domaincombat.EffectStatus || len(plan.Effects[1].Modifiers) != 1 {
			t.Fatalf("template %d plan = %#v", id, plan)
		}
	}
}

func TestTuNhienThuanDonTheUsesLevelHotSequence(t *testing.T) {
	for index, coefficient := range []float64{.25, .45, .65} {
		plan := petActiveSkillPlan("Tự Nhiên Thuẫn Đơn Thể", index+1)
		if plan == nil || len(plan.Effects) != 1 || plan.Effects[0].Periodic == nil {
			t.Fatalf("level %d plan = %#v", index+1, plan)
		}
		if got := plan.Effects[0].Periodic.Coefficient; got != coefficient {
			t.Fatalf("level %d HoT = %.2f, want %.2f", index+1, got, coefficient)
		}
	}
}
