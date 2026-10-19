// Pet turn-start passives apply exact catalog-backed ownership effects before status ticks.
package combat

const diHinhHoanViResistanceFamily = "Di Hình Hoán Vị:kháng debuff"

type diHinhHoanViLevel struct {
	chance     float64
	resistance int
}

var diHinhHoanViLevels = map[int]diHinhHoanViLevel{
	6751: {chance: 0.20, resistance: 80},
	6752: {chance: 0.30, resistance: 120},
	6753: {chance: 0.40, resistance: 150},
}

func (tp *TurnProcessor) activateDiHinhHoanVi(actor *Participant) bool {
	if actor == nil {
		return false
	}
	tp.statusManager.RemoveFamily(actor.ID, diHinhHoanViResistanceFamily)
	level := diHinhHoanViLevel{}
	owned := false
	for skillID, candidate := range diHinhHoanViLevels {
		if actor.SkillIDs[skillID] && (!owned || candidate.resistance > level.resistance) {
			level = candidate
			owned = true
		}
	}
	if !owned || tp.rng.Float64() >= level.chance {
		return false
	}
	tp.statusManager.RemoveAllEffects(actor.ID, PolarityHarmful)
	tp.statusManager.AddEffect(actor.ID, &StatusEffect{
		Duration:   2,
		SourceID:   actor.ID,
		Family:     diHinhHoanViResistanceFamily,
		Polarity:   PolarityBeneficial,
		Removable:  true,
		Modifiers:  []StatModifier{{Stat: StatDebuffResistance, Flat: level.resistance}},
		StackCount: 1,
	})
	return true
}
