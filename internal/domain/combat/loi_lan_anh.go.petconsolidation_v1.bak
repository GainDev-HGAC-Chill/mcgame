// Lôi Lân Ảnh's Sức Mạnh Phán Quyết: on-hit taunt from Lôi Đình Thần Phạt.
package combat

func loiLanAnhTauntEffect(skillID int, skills map[int]bool) (SkillEffect, bool) {
	if skillID < 7038 || skillID > 7046 {
		return SkillEffect{}, false
	}
	chance := 0.0
	switch {
	case skills[7049]:
		chance = 1.50
	case skills[7048]:
		chance = 1.30
	case skills[7047]:
		chance = 1.10
	default:
		return SkillEffect{}, false
	}
	// Existing plannedEffectChance applies debuff hit and resistance modifiers.
	return SkillEffect{
		Kind:      EffectStatus,
		Chance:    chance,
		Duration:  2,
		Polarity:  PolarityHarmful,
		Removable: true,
		Family:    "LoiLanAnh:TraoPhung",
		BuffID:    893,
		Control:   ControlEffect{TauntSourceID: "caster"},
	}, true
}
