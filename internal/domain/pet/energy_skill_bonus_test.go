// Energy-scaled skill tests verify MagicAttack scaling remains idempotent.
package pet

import "testing"

func TestEnergyMagicAttackSkillBonusIsIdempotent(t *testing.T) {
	pet := &Pet{
		Level: 1, GrowRate: 1, AptIntelligence: 1000, AptEnergy: 1000,
		CreatureData: map[string]interface{}{
			"id": 1, "attIntelligence": 10, "attEnergy": 20,
		},
	}
	bonuses := NewSkillStatBonuses()
	bonuses.AddEnergyMagicAttackPerPoint(.15)
	pet.SetSkillBonuses(bonuses)
	pet.RecalculateStats()
	first := petCombatStatIntValue(pet.Property, "finalMAttack")
	energy := petCombatStatIntValue(pet.Property, "finalEnergy")
	pet.RecalculateStats()
	second := petCombatStatIntValue(pet.Property, "finalMAttack")
	if first <= 0 || energy != 20 || second != first {
		t.Fatalf("MagicAttack/Energy = %d/%d then %d", first, energy, second)
	}
}
