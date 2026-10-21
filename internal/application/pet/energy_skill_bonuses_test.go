// Energy passive tests pin Cân Bằng Chi Thược's exact level coefficients.
package pet

import (
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
)

func TestPetEnergyMagicAttackSkillBonuses(t *testing.T) {
	want := map[int]float64{3763: .05, 3771: .10, 3772: .15}
	for skillID, coefficient := range want {
		bonuses := domainpet.NewSkillStatBonuses()
		addPetEnergySkillBonuses(&bonuses, skillID)
		if bonuses.EnergyMagicAttackPerPoint != coefficient {
			t.Fatalf("skill %d coefficient = %.2f, want %.2f", skillID, bonuses.EnergyMagicAttackPerPoint, coefficient)
		}
	}
	bonuses := domainpet.NewSkillStatBonuses()
	addPetEnergySkillBonuses(&bonuses, 999999)
	if !bonuses.IsZero() {
		t.Fatalf("unknown skill bonuses = %#v", bonuses)
	}
}
