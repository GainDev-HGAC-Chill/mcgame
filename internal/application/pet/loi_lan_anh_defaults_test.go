package pet

import (
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

func TestLoiLanAnhHasAllDefaultBaseSkills(t *testing.T) {
	for _, name := range []string{"Lôi Lân Ảnh", "Lôi Lân Ảnh [Tiến Hóa]"} {
		actual := petNamedBaseSkillNames[name]
		wanted := []string{"Lôi Đình Thần Phạt", "Sức Mạnh Phán Quyết", "Huyết Mạch Kỳ Lân"}
		if len(actual) != len(wanted) {
			t.Fatalf("%s has skills %#v", name, actual)
		}
		for i, skill := range wanted {
			if actual[i] != skill {
				t.Fatalf("%s slot %d = %q, want %q", name, i+1, actual[i], skill)
			}
		}
	}
}

func TestLoiLanAnhBloodlineBuffStats(t *testing.T) {
	for _, test := range []struct{ hp, defense, speed float64 }{
		{10, 5, 5}, {20, 10, 10}, {30, 20, 20},
	} {
		buff := &models.BuffTemplate{
			PercentFlag: 1, Prop1: 1, Prop2: 6, Prop3: 11,
			PropNum1: test.hp, PropNum2: test.defense, PropNum3: test.speed,
		}
		bonuses := domainpet.NewSkillStatBonuses()
		addPetSkillBuffBonuses(&bonuses, buff)
		if bonuses.Scaled[1] != test.hp || bonuses.Scaled[6] != test.defense || bonuses.Scaled[11] != test.speed {
			t.Fatalf("Bloodline stats mismatch: expected %v, got %v", test, bonuses)
		}
	}
}
