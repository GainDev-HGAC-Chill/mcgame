// Skill plan executor applies reusable Pet active skill effects without changing playback wire shapes.
package combat

import (
	"math"

	"mcgame-server/internal/domain/skill"
)

func (tp *TurnProcessor) executeSkillPlan(actor *Participant, cmd *BattleCommand, info *SkillInfo) []*BattleAction {
	plan := info.Plan
	if plan == nil {
		return tp.executeAttack(actor, cmd)
	}
	resolver := NewTargetResolverWithRNG(tp.battle, tp.rng)
	targetID := cmd.TargetID
	targets := resolver.ResolveTargets(actor, targetID, plan.Target, plan.Area, plan.TargetCount)
	if len(targets) == 1 && plan.Target == TargetEnemy && plan.Area == AreaNone {
		redirectedID := tp.statusManager.GetConfusionTarget(tp.battle, actor.ID, targets[0].ID, tp.rng)
		redirectedID = tp.statusManager.GetRebellionTarget(tp.battle, actor.ID, redirectedID, tp.rng)
		if redirected := tp.battle.GetParticipant(redirectedID); redirected != nil && redirected.IsAlive {
			targets[0] = redirected
		}
	}
	if len(targets) == 0 {
		return tp.skillMissActions(actor, cmd.SkillID, info.Template)
	}
	if !tp.canPaySkillPlan(actor, targets[0], info) {
		return tp.executeAttack(actor, cmd)
	}
	tp.paySkillPlan(actor, targets[0], info)
	if info.Template.Cooldown > 0 {
		tp.startSkillCooldown(actor.ID, cmd.SkillID, info.Template.Cooldown)
	}
	actions := make([]*BattleAction, 0, len(targets)*2+1)
	for _, target := range targets {
		previousSuccess := true
		for _, effect := range plan.Effects {
			if !target.IsAlive && effect.Kind != EffectRevive {
				break
			}
			if effect.RequiresPreviousSuccess && !previousSuccess {
				continue
			}
			if effect.Chance > 0 && tp.rng.Float64() >= effect.Chance {
				previousSuccess = false
				continue
			}
			effectActions, success := tp.applyPlannedEffect(actor, target, cmd.SkillID, info.Template, effect)
			actions = append(actions, effectActions...)
			previousSuccess = success
		}
	}
	if plan.Sacrifice {
		before := actor.CurrentHP
		actor.CurrentHP = 0
		actor.IsAlive = false
		actions = append(actions, tp.damageAction(actor, actor, cmd.SkillID, info.Template, before, false, false, before))
	}
	if plan.RestTurns > 0 {
		tp.statusManager.AddEffect(actor.ID, &StatusEffect{Family: plan.Family + ":rest", Duration: plan.RestTurns + 1, SourceID: actor.ID, Polarity: PolarityHarmful, Control: ControlEffect{CannotAct: true}, StackCount: 1})
	}
	if shouldTurnBackAfterStationarySkill(info.Template, false) {
		actions = append(actions, buildTurnBackAction(tp.battle.CurrentRound, actor))
	}
	return actions
}

func (tp *TurnProcessor) canPaySkillPlan(actor, target *Participant, info *SkillInfo) bool {
	plan := info.Plan
	if tp.isSkillOnCooldown(actor.ID, info.Template.ID) || actor.CurrentMP < info.Template.MPCost {
		return false
	}
	if plan.RequireHPMax > 0 && float64(actor.CurrentHP) <= float64(actor.MaxHP)*plan.RequireHPMax {
		return false
	}
	for _, cost := range plan.Costs {
		payer := actor
		if plan.CostsOnTarget {
			payer = target
		}
		amount := cost.Fixed
		if cost.Resource == ResourceHP {
			amount += int(math.Round(float64(payer.MaxHP) * cost.Percent))
			if payer.CurrentHP <= amount {
				return false
			}
		} else {
			amount += int(math.Round(float64(payer.MaxMP) * cost.Percent))
			required := amount
			if payer == actor {
				required += info.Template.MPCost
			}
			if payer.CurrentMP < required {
				return false
			}
		}
	}
	return true
}

func (tp *TurnProcessor) paySkillPlan(actor, target *Participant, info *SkillInfo) {
	actor.CurrentMP -= info.Template.MPCost
	for _, cost := range info.Plan.Costs {
		payer := actor
		if info.Plan.CostsOnTarget {
			payer = target
		}
		if cost.Resource == ResourceHP {
			payer.CurrentHP -= cost.Fixed + int(math.Round(float64(payer.MaxHP)*cost.Percent))
		} else {
			payer.CurrentMP -= cost.Fixed + int(math.Round(float64(payer.MaxMP)*cost.Percent))
		}
	}
}

func (tp *TurnProcessor) applyPlannedEffect(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, effect SkillEffect) ([]*BattleAction, bool) {
	switch effect.Kind {
	case EffectPhysicalDamage, EffectMagicDamage:
		return tp.applyPlannedDamage(actor, target, skillID, tpl, effect), true
	case EffectFixedCurrentHPDamage:
		return tp.applyFixedDamage(actor, target, skillID, tpl, int(float64(target.CurrentHP)*effect.Coefficient)), true
	case EffectFixedMaxHPDamage:
		return tp.applyFixedDamage(actor, target, skillID, tpl, int(float64(target.MaxHP)*effect.Coefficient)), true
	case EffectHeal:
		amount := int(float64(tp.statusManager.EffectiveParticipant(actor).MagicAttack) * effect.Coefficient)
		return tp.applyHeal(actor, target, skillID, tpl, amount), true
	case EffectRestoreHP:
		recipient := actor
		if effect.ApplyToTarget {
			recipient = target
		}
		basis := recipient.MaxHP
		if effect.ResourceBasis == ResourceMP {
			basis = recipient.MaxMP
		}
		amount := effect.Fixed + int(float64(basis)*effect.Coefficient)
		return tp.applyHeal(actor, recipient, skillID, tpl, amount), true
	case EffectRestoreMP:
		amount := effect.Fixed + int(float64(actor.MaxMP)*effect.Coefficient)
		actor.CurrentMP = min(actor.MaxMP, actor.CurrentMP+amount)
		return []*BattleAction{tp.stateAction(actor, actor, skillID, tpl, 0)}, true
	case EffectStatus:
		tp.addPlannedStatus(actor, target, skillID, effect)
		return []*BattleAction{tp.stateAction(actor, target, skillID, tpl, 0)}, true
	case EffectRevive:
		if target.IsAlive {
			return nil, false
		}
		target.IsAlive = true
		target.CurrentHP = max(1, int(float64(target.MaxHP)*effect.Coefficient))
		return []*BattleAction{tp.stateAction(actor, target, skillID, tpl, -target.CurrentHP)}, true
	case EffectRemoveOne:
		polarity := PolarityBeneficial
		if target.Side == actor.Side {
			polarity = PolarityHarmful
		}
		removed := tp.statusManager.RemoveRandomEffect(target.ID, polarity, tp.rng)
		return []*BattleAction{tp.stateAction(actor, target, skillID, tpl, 0)}, removed
	case EffectRemoveAll:
		polarity := PolarityBeneficial
		if target.Side == actor.Side {
			polarity = PolarityHarmful
		}
		removed := tp.statusManager.RemoveAllEffects(target.ID, polarity)
		return []*BattleAction{tp.stateAction(actor, target, skillID, tpl, 0)}, removed > 0
	case EffectTransfer:
		return []*BattleAction{tp.applyTransfer(actor, target, skillID, tpl, effect)}, true
	default:
		return nil, false
	}
}

func (tp *TurnProcessor) applyPlannedDamage(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, effect SkillEffect) []*BattleAction {
	damageTemplate := *tpl
	damageTemplate.BaseDamage = 0
	damageTemplate.Multiplier = effect.Coefficient
	if effect.Kind == EffectMagicDamage {
		damageTemplate.SkillType = skill.SkillTypeMagic
	} else {
		damageTemplate.SkillType = skill.SkillTypePhysical
	}
	hitCount := max(1, tpl.AttackCount)
	actions := make([]*BattleAction, 0, hitCount*2)
	for range hitCount {
		if !target.IsAlive {
			target = SelectRandomTarget(tp.battle, getOppositeSide(actor.Side))
			if target == nil {
				break
			}
		}
		before := target.CurrentHP
		effectiveActor := tp.statusManager.EffectiveParticipant(actor)
		effectiveTarget := tp.statusManager.EffectiveParticipant(target)
		var result *DamageResult
		if UsePermilleV2() {
			damageTemplate.AttackCount = 1
			hits := ResolveHits(effectiveActor, effectiveTarget, &SkillInfo{Template: &damageTemplate}, tp.rng)
			result = &DamageResult{}
			if len(hits) > 0 {
				result.Damage = int(hits[0].Damage)
				result.IsCritical = hits[0].IsCrit
				result.IsMiss = hits[0].IsMiss
			}
			if effectiveTarget.DeathImmuneUsed {
				target.DeathImmuneUsed = true
			}
		} else {
			result = CalculatePetSkillDamageWithPosition(effectiveActor, effectiveTarget, &damageTemplate, tp.accuracyModifiers[actor.ID])
		}
		if !result.IsMiss && effect.BonusCurrentHPChance > 0 && tp.rng.Float64() < effect.BonusCurrentHPChance {
			result.Damage += int(float64(before) * effect.BonusCurrentHPRatio)
		}
		if !result.IsMiss {
			result.Damage = tp.applyDamage(target, result.Damage, effect.Kind == EffectMagicDamage)
		}
		action := tp.damageAction(actor, target, skillID, tpl, result.Damage, result.IsCritical, result.IsMiss, before)
		actions = append(actions, action)
		if !result.IsMiss && !target.IsAlive {
			actions = append(actions, tp.deathAction(target))
		}
	}
	return actions
}

func (tp *TurnProcessor) applyFixedDamage(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, amount int) []*BattleAction {
	before := target.CurrentHP
	amount = tp.applyDamage(target, max(0, amount), false)
	actions := []*BattleAction{tp.damageAction(actor, target, skillID, tpl, amount, false, false, before)}
	if !target.IsAlive {
		actions = append(actions, tp.deathAction(target))
	}
	return actions
}

func (tp *TurnProcessor) applyHeal(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, amount int) []*BattleAction {
	before := target.CurrentHP
	target.Heal(max(0, amount))
	actual := target.CurrentHP - before
	return []*BattleAction{tp.stateAction(actor, target, skillID, tpl, -actual)}
}

func (tp *TurnProcessor) addPlannedStatus(actor, target *Participant, skillID int, effect SkillEffect) {
	basis := actor
	if effect.Periodic != nil && effect.Periodic.Basis == 0 && effect.Periodic.Resource != 0 {
		basis = target
	}
	power := periodicBasisValue(tp.statusManager.EffectiveParticipant(basis), effect.Periodic)
	statusType := StatusNormal
	if effect.Control.Sleep {
		statusType = StatusSleep
	} else if effect.Control.Confusion {
		statusType = StatusConfusion
	} else if effect.Control.Stun {
		statusType = StatusDizzy
	}
	control := effect.Control
	if control.TauntSourceID == "caster" {
		control.TauntSourceID = actor.ID
	}
	duration := effect.Duration
	if actor.ID == target.ID {
		duration++
	}
	status := &StatusEffect{Type: statusType, Duration: duration, Power: power, SourceID: actor.ID, TemplateID: skillID, Family: effect.Family, Polarity: effect.Polarity, Removable: effect.Removable, Modifiers: append([]StatModifier(nil), effect.Modifiers...), Periodic: effect.Periodic, Control: control, StackCount: 1}
	if effect.ShieldHits > 0 {
		status.Shield = &ShieldEffect{MagicOnly: effect.ShieldMagicOnly, Pool: int(float64(actor.MaxMP) * effect.ShieldPercentMaxMP), Hits: effect.ShieldHits}
	}
	tp.statusManager.AddEffect(target.ID, status)
}

func periodicBasisValue(actor *Participant, periodic *PeriodicEffect) int {
	if actor == nil || periodic == nil {
		return 0
	}
	switch periodic.Basis {
	case StatAttack:
		return actor.Attack
	case StatMagicAttack:
		return actor.MagicAttack
	case StatEnergy:
		return actor.CurrentSP
	default:
		if periodic.Resource == ResourceMP {
			return actor.MaxMP
		}
		return 0
	}
}

func (tp *TurnProcessor) applyTransfer(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, effect SkillEffect) *BattleAction {
	stats := []CombatStat{StatAttack, StatMagicAttack, StatDefense, StatMagicDefense}
	tp.rng.Shuffle(len(stats), func(i, j int) { stats[i], stats[j] = stats[j], stats[i] })
	count := min(max(1, effect.TransferCount), len(stats))
	positive := make([]StatModifier, 0, count)
	negative := make([]StatModifier, 0, count)
	for _, stat := range stats[:count] {
		positive = append(positive, StatModifier{Stat: stat, Percent: effect.Coefficient})
		negative = append(negative, StatModifier{Stat: stat, Percent: -effect.Coefficient})
	}
	tp.statusManager.AddEffect(actor.ID, &StatusEffect{Duration: effect.Duration + 1, SourceID: actor.ID, TemplateID: skillID, Family: effect.Family + ":gain", Polarity: PolarityBeneficial, Removable: true, Modifiers: positive, StackCount: 1})
	tp.statusManager.AddEffect(target.ID, &StatusEffect{Duration: effect.Duration, SourceID: actor.ID, TemplateID: skillID, Family: effect.Family + ":loss", Polarity: PolarityHarmful, Removable: true, Modifiers: negative, StackCount: 1})
	return tp.stateAction(actor, target, skillID, tpl, 0)
}

func (tp *TurnProcessor) damageAction(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, damage int, critical, miss bool, before int) *BattleAction {
	return &BattleAction{Round: tp.battle.CurrentRound, ActorID: actor.ID, ActorPosition: actor.Position, ActorNewMP: actor.CurrentMP, TargetID: target.ID, TargetPosition: target.Position, ActionType: ActionTypeSkill, BehaviorID: resolveSkillBehaviorID(ActionTypeSkill, tpl), SkillID: skillID, SkillType: tpl.SkillType, Damage: damage, IsCritical: critical, IsMiss: miss, TargetNewHP: target.CurrentHP, TargetDisplayHP: before - damage, HasDisplayHP: !miss, SkillName: tpl.Name, FrontEffID: tpl.FrontEffID, AttackEffID: tpl.AttackEffID, SkillEffID: tpl.SkillEffID, BulletID: tpl.BulletID}
}

func (tp *TurnProcessor) stateAction(actor, target *Participant, skillID int, tpl *skill.SkillTemplate, damage int) *BattleAction {
	return &BattleAction{Round: tp.battle.CurrentRound, ActorID: actor.ID, ActorPosition: actor.Position, ActorNewMP: actor.CurrentMP, TargetID: target.ID, TargetPosition: target.Position, ActionType: ActionTypeSkill, SkillID: skillID, SkillType: tpl.SkillType, Damage: damage, TargetNewHP: target.CurrentHP, TargetNewMP: target.CurrentMP, SkillName: tpl.Name, FrontEffID: tpl.FrontEffID, SkillEffID: tpl.SkillEffID}
}

func (tp *TurnProcessor) deathAction(target *Participant) *BattleAction {
	return &BattleAction{Round: tp.battle.CurrentRound, ActorID: target.ID, ActorPosition: target.Position, TargetID: target.ID, TargetPosition: target.Position, ActionType: ActionTypeDie, IsNPC: target.IsNPC}
}

func (tp *TurnProcessor) skillMissActions(actor *Participant, skillID int, tpl *skill.SkillTemplate) []*BattleAction {
	actions := []*BattleAction{{Round: tp.battle.CurrentRound, ActorID: actor.ID, ActorPosition: actor.Position, TargetPosition: -1, ActionType: ActionTypeSkill, SkillID: skillID, SkillType: tpl.SkillType, SkillName: tpl.Name, IsMiss: true}}
	if shouldTurnBackAfterStationarySkill(tpl, false) {
		actions = append(actions, buildTurnBackAction(tp.battle.CurrentRound, actor))
	}
	return actions
}
