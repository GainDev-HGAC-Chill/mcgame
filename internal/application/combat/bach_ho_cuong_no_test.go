package combat

import (
	domaincombat "mcgame-server/internal/domain/combat"
	"testing"
)

func TestBachHoRushTemplates(t *testing.T) {
	cases := []struct {
		id, level           int
		coefficient, chance float64
	}{
		{7017, 1, 1.20, .20}, {7018, 2, 1.50, .30}, {7019, 3, 1.90, .40},
		{7020, 1, 1.80, .30}, {7021, 2, 2.00, .40}, {7022, 3, 2.20, .45},
		{7023, 1, 2.00, .35}, {7024, 2, 2.20, .45}, {7025, 3, 2.30, .50},
	}
	for _, tc := range cases {
		p := petActiveSkillPlanByTemplateID(tc.id)
		if p == nil || p.Target != domaincombat.TargetEnemy || p.TargetCount != 1 || p.Level != tc.level || len(p.Effects) != 1 {
			t.Fatalf("skill %d: unexpected plan: %#v", tc.id, p)
		}
		e := p.Effects[0]
		if e.Kind != domaincombat.EffectPhysicalDamage || e.Coefficient != tc.coefficient || e.TrueDamageChance != tc.chance || e.TrueDamageAttackRatio != 1 {
			t.Fatalf("skill %d: wrong damage plan %#v", tc.id, e)
		}
	}
}

func TestHoKhieuSonLamDispelsOneBuff(t *testing.T) {
	for level, chance := range []float64{.30, .40, .50} {
		p := petActiveSkillPlan("Hổ Khiếu Sơn Lâm", level+1)
		if p == nil || p.Target != domaincombat.TargetAllEnemy || len(p.Effects) != 2 || p.Effects[1].Kind != domaincombat.EffectRemoveOne || p.Effects[1].Chance != chance {
			t.Fatalf("level %d: invalid dispel: %#v", level+1, p)
		}
	}
}
