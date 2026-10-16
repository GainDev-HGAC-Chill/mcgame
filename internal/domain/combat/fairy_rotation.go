// Open-sourced by BaoLT

package combat

import "go.uber.org/zap"

type FairyActiveEffect struct {
	SkillID   int
	Family    string
	Duration  int
	Supported bool
	Modifiers []StatModifier
}

type FairyRotation struct {
	Slots [5]*FairyActiveEffect
	Next  int
}

func (b *Battle) SetFairyRotation(participantID string, rotation *FairyRotation) {
	if b == nil || participantID == "" || rotation == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.FairyRotations[participantID] = rotation
}

func (tp *TurnProcessor) activateFairyEffect(actor *Participant, cmd *BattleCommand) {
	if tp == nil || tp.battle == nil || actor == nil || cmd == nil || !isFairyAttackTurn(cmd.ActionType) {
		return
	}
	rotation := tp.battle.FairyRotations[actor.ID]
	if rotation == nil {
		return
	}
	slot := rotation.Next % len(rotation.Slots)
	effect := rotation.Slots[slot]
	rotation.Next = (slot + 1) % len(rotation.Slots)
	if effect == nil || effect.SkillID <= 0 {
		return
	}
	if !effect.Supported {
		tp.logger.Warn("fairy active effect unsupported",
			zap.String("actor_id", actor.ID),
			zap.Int("skill_id", effect.SkillID),
			zap.Int("slot", slot))
		return
	}
	tp.statusManager.AddEffect(actor.ID, &StatusEffect{
		Type:       StatusNormal,
		Duration:   effect.Duration,
		SourceID:   actor.ID,
		TemplateID: effect.SkillID,
		Family:     effect.Family,
		Polarity:   PolarityBeneficial,
		Modifiers:  append([]StatModifier(nil), effect.Modifiers...),
		StackCount: 1,
	})
}

func isFairyAttackTurn(actionType int) bool {
	return actionType == ClientActionAttack || actionType == ClientActionSkill || actionType == ClientActionAuto
}
