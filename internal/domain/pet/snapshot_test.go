package pet

import "testing"

func TestCloneDetachesPrivateBonuses(t *testing.T) {
	p := &Pet{
		skillBonuses:     SkillStatBonuses{Scaled: map[int]float64{1: 7}, Direct: map[int]float64{1: 8}},
		equipmentBonuses: EquipmentStatBonuses{Flat: map[int]int{1: 9}, Percent: map[int]int{1: 10}, Float: map[int]float64{1: 11}},
	}
	copy := p.Clone()
	copy.skillBonuses.Scaled[1] = 99
	copy.skillBonuses.Direct[1] = 99
	copy.equipmentBonuses.Flat[1] = 99
	copy.equipmentBonuses.Percent[1] = 99
	copy.equipmentBonuses.Float[1] = 99
	if p.skillBonuses.Scaled[1] != 7 || p.skillBonuses.Direct[1] != 8 || p.equipmentBonuses.Flat[1] != 9 || p.equipmentBonuses.Percent[1] != 10 || p.equipmentBonuses.Float[1] != 11 {
		t.Fatal("clone shared private bonus maps")
	}
}
