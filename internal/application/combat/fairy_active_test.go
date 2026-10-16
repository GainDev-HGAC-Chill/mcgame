// Open-sourced by BaoLT

package combat

import (
	"testing"

	domaincombat "mcgame-server/internal/domain/combat"
	"mcgame-server/internal/gamedata/models"
)

type fakeFairyCombatData struct {
	skills map[int]*models.SkillTemplate
	buffs  map[int]*models.BuffTemplate
}

func (f *fakeFairyCombatData) GetSkill(id int) *models.SkillTemplate { return f.skills[id] }
func (f *fakeFairyCombatData) GetBuff(id int) *models.BuffTemplate   { return f.buffs[id] }

func TestCompileFairyRotationSupportedEffects(t *testing.T) {
	tests := []struct {
		name        string
		skillID     int
		buffID      int
		duration    int
		propID      int
		value       int
		stat        domaincombat.CombatStat
		wantPercent float64
		finalReduce bool
	}{
		{name: "magic attack", skillID: 6231, buffID: 2725, duration: 2, propID: 5, value: 8, stat: domaincombat.StatMagicAttack, wantPercent: .08},
		{name: "physical attack", skillID: 6238, buffID: 2734, duration: 2, propID: 4, value: 13, stat: domaincombat.StatAttack, wantPercent: .13},
		{name: "magic defense", skillID: 6239, buffID: 2735, duration: 2, propID: 7, value: 10, stat: domaincombat.StatMagicDefense, wantPercent: .10},
		{name: "physical defense", skillID: 6243, buffID: 2739, duration: 2, propID: 6, value: 20, stat: domaincombat.StatDefense, wantPercent: .20},
		{name: "speed", skillID: 6262, buffID: 2758, duration: 3, propID: 11, value: 15, stat: domaincombat.StatSpeed, wantPercent: .15},
		{name: "final reduction", skillID: 6271, buffID: 2767, duration: 2, value: 13, stat: domaincombat.StatIncomingDamage, wantPercent: -.13, finalReduce: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, level, _ := fairyActiveFamily(test.skillID)
			data := &fakeFairyCombatData{
				skills: map[int]*models.SkillTemplate{
					test.skillID: {ID: int64(test.skillID), Kind: 1, Level: float64(level), BuffID: float64(test.buffID), BuffRound: float64(test.duration)},
				},
				buffs: map[int]*models.BuffTemplate{},
			}
			buff := &models.BuffTemplate{ID: int64(test.buffID), Level: float64(level), PercentFlag: 1}
			if test.finalReduce {
				buff.EffectNum = float64(test.value)
			} else {
				buff.Prop1 = float64(test.propID)
				buff.PropNum1 = float64(test.value)
			}
			data.buffs[test.buffID] = buff

			rotation, err := compileFairyRotation([]int{test.skillID, -1, -1, -1, -1}, data)
			if err != nil {
				t.Fatalf("compileFairyRotation: %v", err)
			}
			effect := rotation.Slots[0]
			if effect == nil || !effect.Supported || effect.Duration != test.duration || len(effect.Modifiers) != 1 {
				t.Fatalf("effect = %+v", effect)
			}
			modifier := effect.Modifiers[0]
			if modifier.Stat != test.stat || modifier.Percent != test.wantPercent {
				t.Fatalf("modifier = %+v, want stat=%d percent=%v", modifier, test.stat, test.wantPercent)
			}
		})
	}
}

func TestCompileFairyRotationKeepsAmbiguousFamilyUnsupported(t *testing.T) {
	data := &fakeFairyCombatData{
		skills: map[int]*models.SkillTemplate{6227: {ID: 6227, Kind: 1, Level: 1}},
		buffs:  map[int]*models.BuffTemplate{},
	}
	rotation, err := compileFairyRotation([]int{6227, -1, -1, -1, -1}, data)
	if err != nil {
		t.Fatalf("compileFairyRotation: %v", err)
	}
	if effect := rotation.Slots[0]; effect == nil || effect.Supported {
		t.Fatalf("ambiguous effect = %+v, want explicit unsupported", effect)
	}
}

func TestCompileFairyRotationRejectsCatalogDrift(t *testing.T) {
	data := &fakeFairyCombatData{
		skills: map[int]*models.SkillTemplate{6230: {ID: 6230, Kind: 1, Level: 1, BuffID: 2725, BuffRound: 2}},
		buffs:  map[int]*models.BuffTemplate{},
	}
	if _, err := compileFairyRotation([]int{6230, -1, -1, -1, -1}, data); err == nil {
		t.Fatal("catalog drift was accepted")
	}
}
