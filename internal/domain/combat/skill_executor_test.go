// Skill plan tests cover shared damage defenses and representative special mechanics.
package combat

import (
	"math/rand"
	"testing"

	"mcgame-server/internal/domain/skill"
	"mcgame-server/internal/domain/stats"
)

type planSkillGetter struct {
	info *SkillInfo
}

func (g planSkillGetter) GetSkillInfo(int) *SkillInfo {
	return g.info
}

func (g planSkillGetter) GetCreatureSkillIDs(int) []int {
	return nil
}

func TestExecuteSkillPlanPureStatusDealsNoDamage(t *testing.T) {
	battle, actor, target := planTestBattle()
	info := testPlanInfo(&SkillPlan{Family: "support", Target: TargetEnemy, TargetCount: 1, Effects: []SkillEffect{{Kind: EffectStatus, Family: "support", Duration: 2, Polarity: PolarityHarmful, Modifiers: []StatModifier{{Stat: StatDefense, Percent: -.2}}}}})
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	before := target.CurrentHP
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: target.ID, ActionType: ClientActionSkill, SkillID: info.Template.ID})
	if target.CurrentHP != before {
		t.Fatalf("pure status changed HP from %d to %d", before, target.CurrentHP)
	}
	if len(actions) == 0 || !processor.statusManager.HasEffect(target.ID, StatusNormal) {
		t.Fatalf("status was not applied")
	}
}

func TestSharedDamageHonorsInvulnerabilityAndMagicShield(t *testing.T) {
	battle, _, target := planTestBattle()
	processor := NewTurnProcessor(battle, nil)
	processor.statusManager.AddEffect(target.ID, &StatusEffect{Family: "invulnerable", Duration: 2, Control: ControlEffect{Invulnerable: true}})
	if damage := processor.applyDamage(target, 400, false); damage != 0 || target.CurrentHP != target.MaxHP {
		t.Fatalf("invulnerability applied damage=%d hp=%d", damage, target.CurrentHP)
	}
	processor.statusManager.ClearEffects(target.ID)
	processor.statusManager.AddEffect(target.ID, &StatusEffect{Family: "shield", Duration: 4, Shield: &ShieldEffect{MagicOnly: true, Pool: 150, Hits: 3}})
	if damage := processor.applyDamage(target, 100, false); damage != 100 {
		t.Fatalf("physical damage through magic shield = %d, want 100", damage)
	}
	if damage := processor.applyDamage(target, 200, true); damage != 50 {
		t.Fatalf("magic damage after shield absorption = %d, want 50", damage)
	}
}

func TestExecuteSkillPlanRevivesSelectedDeadAlly(t *testing.T) {
	battle, actor, target := planTestBattle()
	target.Side = actor.Side
	target.IsAlive = false
	target.CurrentHP = 0
	info := testPlanInfo(&SkillPlan{Family: "revive", Target: TargetSelectedDeadTeam, TargetCount: 1, Effects: []SkillEffect{{Kind: EffectRevive, Coefficient: .3}}})
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: target.ID, ActionType: ClientActionSkill, SkillID: info.Template.ID})
	if !target.IsAlive || target.CurrentHP != 300 {
		t.Fatalf("revived target alive=%v hp=%d, want true/300", target.IsAlive, target.CurrentHP)
	}
}

func TestTauntForcesSingleEnemyTarget(t *testing.T) {
	battle, actor, target := planTestBattle()
	other := NewParticipant("other", "other", SideEnemy, true)
	other.Position = 2
	other.MaxHP = 1000
	other.CurrentHP = 1000
	battle.AddParticipant(other)
	battle.StatusManager.AddEffect(actor.ID, &StatusEffect{Family: "taunt", Duration: 2, Control: ControlEffect{TauntSourceID: target.ID}})
	resolved := NewTargetResolver(battle).ResolveTargets(actor, other.ID, TargetEnemy, AreaNone, 1)
	if len(resolved) != 1 || resolved[0].ID != target.ID {
		t.Fatalf("taunt target = %#v, want %s", resolved, target.ID)
	}
}

func TestRemoveOneUsesDeterministicEligiblePool(t *testing.T) {
	battle, actor, target := planTestBattle()
	target.Side = actor.Side
	manager := battle.StatusManager
	manager.AddEffect(target.ID, &StatusEffect{Family: "good", Duration: 2, Polarity: PolarityBeneficial, Removable: true})
	manager.AddEffect(target.ID, &StatusEffect{Family: "bad", Duration: 2, Polarity: PolarityHarmful, Removable: true})
	processor := NewTurnProcessor(battle, nil)
	processor.rng = rand.New(rand.NewSource(7))
	_, success := processor.applyPlannedEffect(actor, target, 1, &skill.SkillTemplate{ID: 1}, SkillEffect{Kind: EffectRemoveOne})
	if !success || len(manager.GetEffects(target.ID)) != 1 || manager.GetEffects(target.ID)[0].Family != "good" {
		t.Fatalf("debuff consume result = %#v", manager.GetEffects(target.ID))
	}
}

func TestOwnerResourceConversionPaysAndHealsOwner(t *testing.T) {
	battle := NewBattle(BattleTypePVE, 1)
	battle.Start()
	owner := NewParticipant("owner", "owner", SidePlayer, false)
	owner.EntityType = ParticipantTypeCharacter
	owner.EntityID = 42
	owner.Position = 10
	owner.MaxHP = 1000
	owner.CurrentHP = 500
	owner.MaxMP = 1000
	owner.CurrentMP = 1000
	pet := NewParticipant("pet", "pet", SidePlayer, false)
	pet.EntityType = ParticipantTypePet
	pet.OwnerID = 42
	pet.Position = 15
	pet.MaxMP = 600
	pet.CurrentMP = 600
	battle.AddParticipant(owner)
	battle.AddParticipant(pet)
	plan := &SkillPlan{Family: "convert", Target: TargetSelfPlayer, TargetCount: 1, CostsOnTarget: true, Costs: []SkillCost{{Resource: ResourceMP, Percent: .2}}, Effects: []SkillEffect{{Kind: EffectRestoreHP, Coefficient: .2, ResourceBasis: ResourceMP, ApplyToTarget: true}, {Kind: EffectStatus, Family: "convert", Duration: 5, Periodic: &PeriodicEffect{Resource: ResourceMP, Coefficient: .02}}}}
	info := testPlanInfo(plan)
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	processor.executeSkill(pet, &BattleCommand{ActorID: pet.ID, ActionType: ClientActionSkill, SkillID: info.Template.ID})
	if owner.CurrentMP != 800 || owner.CurrentHP != 700 {
		t.Fatalf("owner hp/mp = %d/%d, want 700/800", owner.CurrentHP, owner.CurrentMP)
	}
	if pet.CurrentMP != 600 {
		t.Fatalf("pet MP = %d, want unchanged 600", pet.CurrentMP)
	}
	effects := processor.statusManager.GetEffects(owner.ID)
	if len(effects) != 1 || effects[0].Power != owner.MaxMP {
		t.Fatalf("periodic MP basis = %#v, want owner max MP %d", effects, owner.MaxMP)
	}
}

func TestTurnStartSeparatesPhysicalAndMagicPeriodicDamage(t *testing.T) {
	participant := NewParticipant("target", "target", SidePlayer, false)
	participant.MaxHP = 1000
	participant.CurrentHP = 1000
	manager := NewStatusEffectManager()
	manager.AddEffect(participant.ID, &StatusEffect{Family: "bleed", Duration: 2, Power: 100, Periodic: &PeriodicEffect{Damage: true, Coefficient: 1}})
	manager.AddEffect(participant.ID, &StatusEffect{Family: "poison", Duration: 2, Power: 200, Periodic: &PeriodicEffect{Damage: true, Magic: true, Coefficient: 1}})
	result := manager.ProcessTurnStart(participant.ID, participant)
	if result.DamageTaken != 300 || result.PhysicalDamageTaken != 100 || result.MagicDamageTaken != 200 {
		t.Fatalf("periodic split = total %d physical %d magic %d", result.DamageTaken, result.PhysicalDamageTaken, result.MagicDamageTaken)
	}
}

func TestPlannedEnemySkillCanRedirectConfusionToAlly(t *testing.T) {
	battle, actor, enemy := planTestBattle()
	ally := NewParticipant("ally", "ally", SidePlayer, false)
	ally.Position = 11
	ally.MaxHP = 1000
	ally.CurrentHP = 1000
	battle.AddParticipant(ally)
	battle.StatusManager.AddEffect(actor.ID, &StatusEffect{Type: StatusConfusion, Duration: 2})
	info := testPlanInfo(&SkillPlan{Family: "confused strike", Target: TargetEnemy, TargetCount: 1, Effects: []SkillEffect{{Kind: EffectFixedMaxHPDamage, Coefficient: .1}}})
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	processor.rng = rand.New(rand.NewSource(2))
	processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: enemy.ID, ActionType: ClientActionSkill, SkillID: info.Template.ID})
	if actor.CurrentHP+ally.CurrentHP != 1900 {
		t.Fatalf("confused side total HP = %d, want 1900", actor.CurrentHP+ally.CurrentHP)
	}
	if enemy.CurrentHP != enemy.MaxHP {
		t.Fatalf("intended enemy HP = %d, want unchanged %d", enemy.CurrentHP, enemy.MaxHP)
	}
}

func TestEffectiveParticipantUpdatesDamageCutForReductionModifiers(t *testing.T) {
	participant := NewParticipant("target", "target", SidePlayer, false)
	participant.MaxHP = 1000
	participant.CurrentHP = 1000
	participant.ReduceHurt1 = 20
	participant.ReduceHurt2 = 30
	participant.EffStats = shimEffStats(participant)
	manager := NewStatusEffectManager()
	manager.AddEffect(participant.ID, &StatusEffect{Family: "weaken", Duration: 2, Modifiers: []StatModifier{{Stat: StatReducePhysical, Flat: -10}, {Stat: StatReduceMagic, Flat: -20}}})
	effective := manager.EffectiveParticipant(participant)
	if got := effective.EffStats.Get(stats.PropPhysDmgCut); got != 100 {
		t.Fatalf("physical damage cut = %d, want 100 per-mille", got)
	}
	if got := effective.EffStats.Get(stats.PropMagDmgCut); got != 100 {
		t.Fatalf("magic damage cut = %d, want 100 per-mille", got)
	}
	if got := effective.EffStats.Get(stats.PropFinalPhysDmgDown); got != 0 {
		t.Fatalf("final physical damage down = %d, want unchanged 0", got)
	}
}

func planTestBattle() (*Battle, *Participant, *Participant) {
	battle := NewBattle(BattleTypePVE, 1)
	battle.Start()
	actor := NewParticipant("actor", "actor", SidePlayer, false)
	actor.Position = 10
	actor.Level = 10
	actor.MaxHP = 1000
	actor.CurrentHP = 1000
	actor.MaxMP = 1000
	actor.CurrentMP = 1000
	actor.Attack = 200
	actor.MagicAttack = 200
	target := NewParticipant("target", "target", SideEnemy, true)
	target.Position = 0
	target.Level = 10
	target.MaxHP = 1000
	target.CurrentHP = 1000
	target.Defense = 100
	target.MagicDefense = 100
	battle.AddParticipant(actor)
	battle.AddParticipant(target)
	return battle, actor, target
}

func testPlanInfo(plan *SkillPlan) *SkillInfo {
	return &SkillInfo{Template: &skill.SkillTemplate{ID: 9001, Name: plan.Family, SkillType: skill.SkillTypeBuff, Range: 2, AttackCount: 1}, Plan: plan}
}
