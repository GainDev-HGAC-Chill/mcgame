// Open-sourced by BaoLT

package pet

import domainpet "mcgame-server/internal/domain/pet"

type petPotentialBonus struct {
	propID      int
	coefficient float64
}

var petPotentialBonuses = map[int][]petPotentialBonus{
	3744: {{1, 450}, {2, 300}, {6, 180}, {7, 180}},
	3745: {{1, 640}, {2, 430}, {6, 290}, {7, 290}},
	3746: {{1, 830}, {2, 560}, {6, 400}, {7, 400}},
	3747: {{1, 150}, {15, 0.55}, {16, 0.55}, {10, 1}},
	3748: {{1, 220}, {15, 0.9}, {16, 0.9}, {10, 1.6}},
	3749: {{1, 290}, {15, 1.25}, {16, 1.25}, {10, 2.2}},
	3750: {{11, 50}, {8, 1}, {9, 1}, {12, 0.55}},
	3751: {{2, 270}, {4, 160}, {5, 160}, {13, 0.55}},
	3752: {{11, 70}, {8, 1.6}, {9, 1.6}, {12, 0.9}},
	3753: {{2, 390}, {4, 220}, {5, 220}, {13, 0.9}},
	3754: {{11, 90}, {8, 2.2}, {9, 2.2}, {12, 1.25}},
	3755: {{2, 510}, {4, 290}, {5, 290}, {13, 1.25}},
}

func addPetPotentialSkillBonuses(bonuses *domainpet.SkillStatBonuses, skillID int, quality float64) {
	if bonuses == nil || quality <= 0 {
		return
	}
	for _, bonus := range petPotentialBonuses[skillID] {
		bonuses.AddDirect(bonus.propID, bonus.coefficient*quality)
	}
}
