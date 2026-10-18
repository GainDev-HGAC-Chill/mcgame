// Open-sourced by BaoLT

package pet

import (
	"math"
	"strconv"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
)

func TestAddPetPotentialSkillBonuses_ScalesCheckedFamiliesByQuality(t *testing.T) {
	tests := []struct {
		skillID int
		propID  int
		want    float64
	}{
		{3744, 1, 900},
		{3745, 6, 580},
		{3746, 7, 800},
		{3747, 10, 2},
		{3748, 15, 1.8},
		{3749, 16, 2.5},
		{3750, 11, 100},
		{3751, 4, 320},
		{3752, 8, 3.2},
		{3753, 5, 440},
		{3754, 12, 2.5},
		{3755, 2, 1020},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.skillID), func(t *testing.T) {
			bonuses := domainpet.NewSkillStatBonuses()
			addPetPotentialSkillBonuses(&bonuses, tt.skillID, 2)
			if got := bonuses.Direct[tt.propID]; got != tt.want {
				t.Fatalf("skill %d prop %d = %v, want %v", tt.skillID, tt.propID, got, tt.want)
			}
		})
	}
}

func TestAddPetPotentialSkillBonuses_BloodLevelThree(t *testing.T) {
	bonuses := domainpet.NewSkillStatBonuses()
	addPetPotentialSkillBonuses(&bonuses, 3755, 2.33)

	wants := map[int]float64{
		2:  1188.3,
		4:  675.7,
		5:  675.7,
		13: 2.9125,
	}
	for propID, want := range wants {
		if got := bonuses.Direct[propID]; math.Abs(got-want) > 0.000001 {
			t.Errorf("prop %d = %v, want %v", propID, got, want)
		}
	}
}
