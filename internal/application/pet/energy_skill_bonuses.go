// Energy-scaled Pet passives use final Energy without recursive stat compounding.
package pet

import domainpet "mcgame-server/internal/domain/pet"

var petEnergyMagicAttackBonuses = map[int]float64{
	3763: 0.05,
	3771: 0.10,
	3772: 0.15,
}

func addPetEnergySkillBonuses(bonuses *domainpet.SkillStatBonuses, skillID int) {
	if bonuses == nil {
		return
	}
	bonuses.AddEnergyMagicAttackPerPoint(petEnergyMagicAttackBonuses[skillID])
}
