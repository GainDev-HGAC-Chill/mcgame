// Open-sourced by BaoLT

package pet

import (
	"testing"

	"mcgame-server/internal/gamedata/models"
)

func TestPetRecalculateStats_UsesPersistedAptitude(t *testing.T) {
	template := &models.CreatureTemplate{
		ID:          1,
		Name:        "persisted-aptitude",
		AttStrength: 10,
		AptStrength: 1000,
		GrowBase:    1,
	}

	p := NewPetFromCreature(1, template)
	p.AptStrength = 2000
	p.RecalculateStats()

	if got := p.Property["finalAttack"]; got != 57 {
		t.Fatalf("finalAttack = %#v, want 57 from persisted AptStrength=2000", got)
	}
}

func TestPetRecalculateStats_IncludesCreatureDirectCombatProps(t *testing.T) {
	template := &models.CreatureTemplate{
		ID:           2,
		Name:         "direct-props",
		GrowBase:     1,
		PropHit:      30,
		PropDodge:    10,
		PropSpeed:    100,
		PropCritical: 35,
		PropCombo:    15,
		PropCounter:  20,
		PropDefy:     25,
		PropReborn:   5,
		ResiDefy:     40,
	}

	p := NewPetFromCreature(1, template)

	wants := map[string]int{
		"finalHit":        130,
		"finalDodge":      10,
		"finalSpeed":      200,
		"finalCritical":   35,
		"finalCombo":      15,
		"finalCounter":    20,
		"finalDefy":       25,
		"finalRebornRate": 5,
		"finalResiDefy":   40,
	}
	for key, want := range wants {
		if got := p.Property[key]; got != want {
			t.Errorf("property[%q] = %#v, want %d", key, got, want)
		}
	}
}
