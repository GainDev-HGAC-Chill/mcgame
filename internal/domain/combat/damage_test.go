// Open-sourced by BaoLT

// Damage formula tests for crit damage normalization.
package combat

import (
	"testing"

	"mcgame-server/internal/domain/skill"
)

func TestCalculatePetSkillDamageUsesAttackMultiplierWithoutLevelBonus(t *testing.T) {
	attacker := NewParticipant("attacker", "attacker", SidePlayer, false)
	attacker.Level = 100
	attacker.Attack = 200
	attacker.Hit = 100
	attacker.Position = 15
	defender := NewParticipant("defender", "defender", SideEnemy, true)
	defender.MaxHP = 10000
	defender.CurrentHP = defender.MaxHP
	defender.Position = 5
	tpl := &skill.SkillTemplate{SkillType: skill.SkillTypePhysical, Multiplier: 1, Range: 2}
	for range 50 {
		result := CalculatePetSkillDamageWithPosition(attacker, defender, tpl)
		if result.IsMiss {
			t.Fatal("guaranteed hit missed")
		}
		if result.Damage < 160 || result.Damage > 250 {
			t.Fatalf("damage = %d, want attack-based result without level multiplier", result.Damage)
		}
	}
}

func TestCriticalDamageMultiplier_UsesHiddenBaseAndSeparateBonus(t *testing.T) {
	if got := criticalDamageMultiplier(BaseCriticalDamage); got != 1.5 {
		t.Fatalf("criticalDamageMultiplier(150) = %v, want 1.5", got)
	}
	if got := criticalDamageMultiplier(25); got != 1.875 {
		t.Fatalf("criticalDamageMultiplier(25) = %v, want 1.875", got)
	}
	if got := criticalDamageMultiplier(175); got != 1.875 {
		t.Fatalf("criticalDamageMultiplier(175) = %v, want 1.875", got)
	}
}
