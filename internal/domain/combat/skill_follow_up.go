// Skill follow-up helpers execute exact on-kill plans and resistance-aware status rolls.
package combat

import "mcgame-server/internal/domain/stats"

func (tp *TurnProcessor) executeOnKillFollowUp(actor *Participant, followUp *SkillOnKillFollowUp) []*BattleAction {
	if actor == nil || followUp == nil || tp.skillGetter == nil {
		return nil
	}
	targetCount := 0
	for passiveID, count := range followUp.PassiveTargetCounts {
		if actor.SkillIDs[passiveID] && count > targetCount {
			targetCount = count
		}
	}
	if targetCount <= 0 {
		return nil
	}
	templateID := followUp.TemplateIDs[targetCount]
	info := tp.skillGetter.GetSkillInfo(templateID)
	if info == nil || info.Template == nil || info.Plan == nil || len(info.Plan.Effects) == 0 {
		return nil
	}
	resolver := NewTargetResolverWithRNG(tp.battle, tp.rng)
	actions := make([]*BattleAction, 0, targetCount*3)
	wave := 0
	for {
		targets := resolver.ResolveTargets(actor, "", followUp.Target, AreaNone, targetCount)
		if len(targets) == 0 {
			break
		}
		if wave > 0 {
			actions = append(actions, &BattleAction{Round: tp.battle.CurrentRound, ActorID: actor.ID, ActorPosition: actor.Position, TargetID: actor.ID, TargetPosition: actor.Position, ActionType: ActionTypeAuto, Effects: []string{"skip"}})
		}
		wave++
		waveKilledTarget := false
		for _, target := range targets {
			wasAlive := target.IsAlive
			previousSuccess := true
			for _, effect := range info.Plan.Effects {
				if !target.IsAlive && effect.Kind != EffectRevive {
					break
				}
				if effect.RequiresPreviousSuccess && !previousSuccess {
					continue
				}
				if !tp.plannedEffectSucceeds(actor, target, effect) {
					previousSuccess = false
					continue
				}
				effectActions, success := tp.applyPlannedEffect(actor, target, templateID, info.Template, effect)
				actions = append(actions, effectActions...)
				previousSuccess = success
			}
			if wasAlive && !target.IsAlive {
				waveKilledTarget = true
			}
		}
		if !waveKilledTarget {
			break
		}
	}
	return actions
}

func (tp *TurnProcessor) plannedEffectChance(actor, target *Participant, effect SkillEffect) float64 {
	chance := effect.Chance
	if effect.Kind != EffectStatus || effect.Polarity != PolarityHarmful {
		return chance
	}
	if chance <= 0 {
		chance = 1
	}
	attacker := tp.statusManager.EffectiveParticipant(actor)
	defender := tp.statusManager.EffectiveParticipant(target)
	chance += float64(attacker.EffStats.Get(stats.PropDebuffHit)-defender.EffStats.Get(stats.PropResistDebuff)) / float64(stats.PerMille)
	accuracyProp, resistanceProp := plannedControlProps(effect.Control)
	if accuracyProp != 0 && resistanceProp != 0 {
		chance += float64(attacker.EffStats.Get(accuracyProp)-defender.EffStats.Get(resistanceProp)) / float64(stats.PerMille)
	}
	return min(1.0, max(0.0, chance))
}

func plannedControlProps(control ControlEffect) (stats.PropID, stats.PropID) {
	switch {
	case control.Stun:
		return stats.PropStunAcc, stats.PropResistStun
	case control.Confusion:
		return stats.PropConfuseAcc, stats.PropResistConfuse
	case control.Sleep:
		return stats.PropSleepAcc, stats.PropResistSleep
	default:
		return 0, 0
	}
}

func (tp *TurnProcessor) plannedEffectSucceeds(actor, target *Participant, effect SkillEffect) bool {
	if effect.Chance <= 0 && (effect.Kind != EffectStatus || effect.Polarity != PolarityHarmful) {
		return true
	}
	return tp.rng.Float64() < tp.plannedEffectChance(actor, target, effect)
}
