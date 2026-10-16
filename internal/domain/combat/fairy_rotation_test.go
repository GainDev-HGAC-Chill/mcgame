// Open-sourced by BaoLT

package combat

import "testing"

func TestFairyRotationAdvancesOnlyOnAttackTurns(t *testing.T) {
	battle := NewBattle(BattleTypePVE, 1)
	actor := NewParticipant("player", "Player", SidePlayer, false)
	actor.Attack = 100
	battle.AddParticipant(actor)
	rotation := &FairyRotation{}
	rotation.Slots[0] = &FairyActiveEffect{
		SkillID:   6230,
		Family:    "fairy:6230",
		Duration:  2,
		Supported: true,
		Modifiers: []StatModifier{{Stat: StatAttack, Percent: .10}},
	}
	rotation.Slots[1] = &FairyActiveEffect{SkillID: 6227, Family: "fairy:6227"}
	battle.SetFairyRotation(actor.ID, rotation)
	processor := NewTurnProcessor(battle, nil)

	processor.activateFairyEffect(actor, &BattleCommand{ActionType: ClientActionDefend})
	if rotation.Next != 0 || len(battle.StatusManager.GetEffects(actor.ID)) != 0 {
		t.Fatalf("defend advanced rotation: next=%d effects=%+v", rotation.Next, battle.StatusManager.GetEffects(actor.ID))
	}

	processor.activateFairyEffect(actor, &BattleCommand{ActionType: ClientActionAttack})
	if rotation.Next != 1 {
		t.Fatalf("attack next = %d, want 1", rotation.Next)
	}
	effective := battle.StatusManager.EffectiveParticipant(actor)
	if effective.Attack != 110 {
		t.Fatalf("effective attack = %d, want 110", effective.Attack)
	}

	processor.activateFairyEffect(actor, &BattleCommand{ActionType: ClientActionSkill})
	if rotation.Next != 2 {
		t.Fatalf("unsupported slot next = %d, want 2", rotation.Next)
	}
	if len(battle.StatusManager.GetEffects(actor.ID)) != 1 {
		t.Fatalf("unsupported effect mutated statuses: %+v", battle.StatusManager.GetEffects(actor.ID))
	}
}

func TestFairyFinalReductionUsesIncomingDamageModifier(t *testing.T) {
	battle := NewBattle(BattleTypePVE, 1)
	actor := NewParticipant("player", "Player", SidePlayer, false)
	battle.AddParticipant(actor)
	rotation := &FairyRotation{}
	rotation.Slots[0] = &FairyActiveEffect{
		SkillID:   6271,
		Family:    "fairy:6269",
		Duration:  2,
		Supported: true,
		Modifiers: []StatModifier{{Stat: StatIncomingDamage, Percent: -.13}},
	}
	battle.SetFairyRotation(actor.ID, rotation)
	processor := NewTurnProcessor(battle, nil)
	processor.activateFairyEffect(actor, &BattleCommand{ActionType: ClientActionAuto})

	if got := battle.StatusManager.ApplyDamage(actor.ID, 100, false); got != 87 {
		t.Fatalf("reduced damage = %d, want 87", got)
	}
}
