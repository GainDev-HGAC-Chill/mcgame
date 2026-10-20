package combat

import (
	domaincombat "mcgame-server/internal/domain/combat"
	"testing"
)

func TestLoiLanAnhExactSkillTemplates(t *testing.T) {
	cases := []struct {
		id, level                          int
		coefficient, chance, maxHP, repeat float64
	}{
		{7038, 1, 1.40, .10, .10, 0}, {7039, 2, 1.50, .10, .10, 0}, {7040, 3, 1.70, .10, .10, 0},
		{7041, 1, 1.80, .15, .15, 0}, {7042, 2, 1.90, .20, .20, 0}, {7043, 3, 2.00, .30, .30, 0},
		{7044, 1, 2.10, .30, .30, .05}, {7045, 2, 2.20, .30, .30, .10}, {7046, 3, 2.40, .30, .30, .15},
	}
	for _, tc := range cases {
		plan := petActiveSkillPlanByTemplateID(tc.id)
		if plan == nil || plan.Level != tc.level || plan.Target != domaincombat.TargetEnemy || plan.TargetCount != 1 || len(plan.Effects) != 1 || plan.RepeatChance != tc.repeat {
			t.Fatalf("ID %d: invalid plan: %#v", tc.id, plan)
		}
		effect := plan.Effects[0]
		if effect.Kind != domaincombat.EffectPhysicalDamage || effect.Coefficient != tc.coefficient || effect.BonusCasterMaxHPChance != tc.chance || effect.BonusCasterMaxHPRatio != tc.maxHP {
			t.Fatalf("ID %d: damage plan invalid: %#v", tc.id, effect)
		}
	}
}
