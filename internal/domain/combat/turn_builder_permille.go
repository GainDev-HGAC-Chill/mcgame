// Open-sourced by BaoLT

// Per-mille damage action builder. Calls ResolveHits instead of CalculateDamageWithPosition.
package combat

import (
	"go.uber.org/zap"

	"mcgame-server/internal/domain/skill"
)

func (tp *TurnProcessor) buildDamageActionsPermille(
	actor, target *Participant,
	actionType ActionType,
	skillID int,
	actorNewMP int,
	needsMovement bool,
	allowCounter bool,
	sk *SkillInfo,
) []*BattleAction {
	actions := make([]*BattleAction, 0, 4+maxComboHits*4)

	skillName := ""
	if actionType == ActionTypeSkill && sk != nil && sk.Template != nil {
		skillName = sk.Template.Name
	}
	skillBehaviorID := resolveSkillBehaviorIDFromInfo(actionType, sk)

	if needsMovement {
		actions = append(actions, &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        actor.ID,
			ActorPosition:  actor.Position,
			ActorNewMP:     actorNewMP,
			TargetID:       target.ID,
			TargetPosition: target.Position,
			ActionType:     ActionTypeTurnTo,
			SkillName:      skillName,
			FrontEffID:     skillFrontEffectFromInfo(sk),
		})
		actions = append(actions, &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        actor.ID,
			ActorPosition:  actor.Position,
			ActorNewMP:     actorNewMP,
			TargetID:       target.ID,
			TargetPosition: target.Position,
			ActionType:     ActionTypeMoveToTarget,
		})
	}

	effectiveActor := tp.statusManager.EffectiveParticipant(actor)
	effectiveTarget := tp.statusManager.EffectiveParticipant(target)
	hits := ResolveHits(effectiveActor, effectiveTarget, sk, tp.rng)
	if effectiveTarget.DeathImmuneUsed {
		target.DeathImmuneUsed = true
	}

	for i, h := range hits {
		if !actor.IsAlive || !target.IsAlive {
			break
		}

		hitAction := &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        actor.ID,
			ActorPosition:  actor.Position,
			ActorNewMP:     actorNewMP,
			TargetID:       target.ID,
			TargetPosition: target.Position,
			ActionType:     actionType,
			BehaviorID:     skillBehaviorID,
			SkillID:        skillID,
			SkillType:      skillTypeFromInfo(sk),
			Damage:         int(h.Damage),
			IsCritical:     h.IsCrit,
			IsMiss:         h.IsMiss,
			AttackEffID:    resolveActionAttackEffectFromInfo(actionType, sk),
			BulletID:       skillBulletEffectFromInfo(sk),
		}
		if i == 0 && !needsMovement {
			hitAction.SkillName = skillName
			hitAction.FrontEffID = skillFrontEffectFromInfo(sk)
		}

		if !h.IsMiss {
			preDamageHP := target.CurrentHP
			damage := int(h.Damage)
			if h.IsDeathSave {
				damage = 0
			}
			if tp.defendingIDs[target.ID] {
				damage /= 2
			}
			magic := sk != nil && sk.Template != nil && sk.Template.SkillType == skill.SkillTypeMagic
			damage = tp.applyDamage(target, damage, magic)
			hitAction.Damage = damage
			hitAction.TargetNewHP = target.CurrentHP
			hitAction.TargetDisplayHP = preDamageHP - damage
			hitAction.HasDisplayHP = true
			hitAction.SkillEffID = skillImpactEffectFromInfo(sk)
			tp.logger.Debug("Per-mille hit",
				zap.String("actor_id", actor.ID),
				zap.String("target_id", target.ID),
				zap.Int("damage", damage),
				zap.Bool("crit", h.IsCrit),
				zap.Bool("bypass", h.IsDefBypass),
				zap.Int("target_hp", target.CurrentHP))
		}
		actions = append(actions, hitAction)

		if h.IsMiss {
			continue
		}

		hurtType := ActionTypeHurt
		if tp.defendingIDs[target.ID] {
			hurtType = ActionTypeDefended
		}
		actions = append(actions, &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        target.ID,
			ActorPosition:  target.Position,
			TargetID:       actor.ID,
			TargetPosition: actor.Position,
			ActionType:     hurtType,
		})

		if !target.IsAlive {
			actions = append(actions, &BattleAction{
				Round:          tp.battle.CurrentRound,
				ActorID:        target.ID,
				ActorPosition:  target.Position,
				TargetID:       target.ID,
				TargetPosition: target.Position,
				ActionType:     ActionTypeDie,
				IsNPC:          target.IsNPC,
			})
			continue
		}

		actions = append(actions, &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        target.ID,
			ActorPosition:  target.Position,
			TargetID:       target.ID,
			TargetPosition: target.Position,
			ActionType:     ActionTypeIdle,
		})
		if hitAction.Damage > 0 {
			actions = append(actions, tp.petReflectActions(actor, target, hitAction.Damage)...)
		}

		if allowCounter && target.IsAlive && actor.IsAlive && h.Flags.Counter {
			tp.logger.Debug("Counter triggered (permille)",
				zap.String("target_id", target.ID))
			actions = append(actions, tp.buildCounterActions(target, actor)...)
		}
	}

	if needsMovement {
		actions = append(actions, &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        actor.ID,
			ActorPosition:  actor.Position,
			TargetID:       actor.ID,
			TargetPosition: actor.Position,
			ActionType:     ActionTypeBack,
		})
		actions = append(actions, &BattleAction{
			Round:          tp.battle.CurrentRound,
			ActorID:        actor.ID,
			ActorPosition:  actor.Position,
			TargetID:       actor.ID,
			TargetPosition: actor.Position,
			ActionType:     ActionTypeTurnBack,
		})
	}

	return actions
}

func resolveSkillBehaviorIDFromInfo(actionType ActionType, sk *SkillInfo) int {
	if actionType != ActionTypeSkill || sk == nil || sk.Template == nil {
		return 0
	}
	return resolveSkillBehaviorID(actionType, sk.Template)
}

func skillTypeFromInfo(sk *SkillInfo) skill.SkillType {
	if sk == nil || sk.Template == nil {
		return 0
	}
	return sk.Template.SkillType
}

func resolveActionAttackEffectFromInfo(actionType ActionType, sk *SkillInfo) int {
	if sk == nil || sk.Template == nil {
		return 0
	}
	return resolveActionAttackEffect(actionType, sk.Template)
}

func skillFrontEffectFromInfo(sk *SkillInfo) int {
	if sk == nil || sk.Template == nil {
		return 0
	}
	return skillFrontEffect(sk.Template)
}

func skillBulletEffectFromInfo(sk *SkillInfo) int {
	if sk == nil || sk.Template == nil {
		return 0
	}
	return skillBulletEffect(sk.Template)
}

func skillImpactEffectFromInfo(sk *SkillInfo) int {
	if sk == nil || sk.Template == nil {
		return 0
	}
	return skillImpactEffect(sk.Template)
}
