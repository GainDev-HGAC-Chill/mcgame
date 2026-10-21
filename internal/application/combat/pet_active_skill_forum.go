// Forum-audited exact Pet plans cover special-source skills corroborated by seed data.
package combat

import domaincombat "mcgame-server/internal/domain/combat"

func forumDamagePlan(family string, level int, kind domaincombat.EffectKind, target domaincombat.TargetType, area domaincombat.AreaType, targetCount int, coefficient float64, restTurns int) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family: family, Level: level, Target: target, Area: area, TargetCount: targetCount, RestTurns: restTurns,
		Effects: []domaincombat.SkillEffect{{Kind: kind, Coefficient: coefficient}},
	}
}

func forumDamageStatusPlan(family string, level int, kind domaincombat.EffectKind, coefficient, chance float64, duration, buffID int, stat domaincombat.CombatStat, percent float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family: family, Level: level, Target: domaincombat.TargetEnemy, TargetCount: 1,
		Effects: []domaincombat.SkillEffect{
			{Kind: kind, Coefficient: coefficient},
			forumStatusEffect(family, chance, duration, buffID, domaincombat.ControlEffect{}, domaincombat.StatModifier{Stat: stat, Percent: percent}),
		},
	}
}

func forumStatusPlan(family string, level int, target domaincombat.TargetType, area domaincombat.AreaType, targetCount int, chance float64, duration, buffID int, stat domaincombat.CombatStat, percent float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family: family, Level: level, Target: target, Area: area, TargetCount: targetCount,
		Effects: []domaincombat.SkillEffect{forumStatusEffect(family, chance, duration, buffID, domaincombat.ControlEffect{}, domaincombat.StatModifier{Stat: stat, Percent: percent})},
	}
}

func forumControlPlan(family string, level int, target domaincombat.TargetType, area domaincombat.AreaType, targetCount int, chance float64, duration, buffID int, control domaincombat.ControlEffect) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family: family, Level: level, Target: target, Area: area, TargetCount: targetCount,
		Effects: []domaincombat.SkillEffect{forumStatusEffect(family, chance, duration, buffID, control)},
	}
}

func forumStatusEffect(family string, chance float64, duration, buffID int, control domaincombat.ControlEffect, modifiers ...domaincombat.StatModifier) domaincombat.SkillEffect {
	return domaincombat.SkillEffect{
		Kind: domaincombat.EffectStatus, Chance: chance, Duration: duration, Polarity: domaincombat.PolarityHarmful,
		Removable: true, Family: family, BuffID: buffID, Control: control, Modifiers: modifiers,
	}
}
