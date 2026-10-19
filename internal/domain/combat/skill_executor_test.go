// Skill plan tests cover shared damage defenses and representative special mechanics.
package combat

import (
	"math/rand"
	"strconv"
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

type mappedPlanSkillGetter map[int]*SkillInfo

func (g mappedPlanSkillGetter) GetSkillInfo(id int) *SkillInfo {
	return g[id]
}

func (g mappedPlanSkillGetter) GetCreatureSkillIDs(int) []int {
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

func TestBachBienKillTriggersVoSongOnThreeRandomLivingTargets(t *testing.T) {
	battle, actor, killed := planTestBattle()
	actor.SkillIDs[6723] = true
	killed.CurrentHP = 1
	for index, position := range []int{2, 4, 6} {
		target := NewParticipant("follow-"+strconv.Itoa(index), "follow", SideEnemy, true)
		target.Position = position
		target.MaxHP = 5000
		target.CurrentHP = 5000
		target.Defense = 100
		battle.AddParticipant(target)
	}
	primaryPlan := &SkillPlan{
		Family:      "Bách Biến Trùng Kích",
		Target:      TargetEnemy,
		TargetCount: 1,
		Effects:     []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: 1.4}},
		OnKill: &SkillOnKillFollowUp{
			PassiveTargetCounts: map[int]int{6721: 1, 6722: 2, 6723: 3},
			TemplateIDs:         map[int]int{1: 6724, 2: 6725, 3: 6726},
			Target:              TargetRandomEnemy,
		},
	}
	primary := &SkillInfo{Template: &skill.SkillTemplate{ID: 6712, Name: "Bách Biến Trùng Kích", SkillType: skill.SkillTypePhysical, AttackCount: 1, Range: 1}, Plan: primaryPlan}
	follow := testFollowUpInfo(6726, 2.5)
	processor := NewTurnProcessor(battle, mappedPlanSkillGetter{6712: primary, 6726: follow})
	processor.rng = rand.New(rand.NewSource(7))
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: killed.ID, ActionType: ClientActionSkill, SkillID: 6712})

	if killed.IsAlive {
		t.Fatal("Bách Biến target survived, want on-kill trigger fixture")
	}
	followTargets := make(map[string]bool)
	for _, action := range actions {
		if action.ActionType == ActionTypeSkill && action.SkillID == 6726 && action.Damage > 0 {
			followTargets[action.TargetID] = true
		}
	}
	if len(followTargets) != 3 {
		t.Fatalf("follow-up targets = %v, want three distinct living enemies", followTargets)
	}
	if followTargets[killed.ID] {
		t.Fatalf("follow-up targeted defeated participant %q", killed.ID)
	}
}

func TestBachBienDoesNotTriggerVoSongWithoutKillOrPassive(t *testing.T) {
	build := func(ownsPassive bool, targetHP int) int {
		battle, actor, target := planTestBattle()
		if ownsPassive {
			actor.SkillIDs[6723] = true
		}
		target.CurrentHP = targetHP
		plan := &SkillPlan{
			Family:      "Bách Biến Trùng Kích",
			Target:      TargetEnemy,
			TargetCount: 1,
			Effects:     []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: 1.4}},
			OnKill: &SkillOnKillFollowUp{
				PassiveTargetCounts: map[int]int{6723: 3},
				TemplateIDs:         map[int]int{3: 6726},
				Target:              TargetRandomEnemy,
			},
		}
		primary := &SkillInfo{Template: &skill.SkillTemplate{ID: 6712, Name: "Bách Biến Trùng Kích", SkillType: skill.SkillTypePhysical, AttackCount: 1}, Plan: plan}
		follow := testFollowUpInfo(6726, 2.5)
		processor := NewTurnProcessor(battle, mappedPlanSkillGetter{6712: primary, 6726: follow})
		actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: target.ID, ActionType: ClientActionSkill, SkillID: 6712})
		count := 0
		for _, action := range actions {
			if action.SkillID == 6726 {
				count++
			}
		}
		return count
	}
	if count := build(true, 5000); count != 0 {
		t.Fatalf("non-lethal Bách Biến emitted %d follow-up actions", count)
	}
	if count := build(false, 1); count != 0 {
		t.Fatalf("Bách Biến without Vô Song emitted %d follow-up actions", count)
	}
}

func TestVoSongChainsAnotherWaveWhenFollowUpKills(t *testing.T) {
	battle, actor, killed := planTestBattle()
	actor.SkillIDs[6723] = true
	actor.Attack = 1000
	killed.CurrentHP = 1
	for index, position := range []int{1, 2, 3, 4, 5} {
		target := NewParticipant("chain-"+strconv.Itoa(index), "chain", SideEnemy, true)
		target.Position = position
		target.MaxHP = 10
		target.CurrentHP = 10
		battle.AddParticipant(target)
	}
	plan := &SkillPlan{
		Family:      "Bách Biến Trùng Kích",
		Target:      TargetEnemy,
		TargetCount: 1,
		Effects:     []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: 1.4}},
		OnKill: &SkillOnKillFollowUp{
			PassiveTargetCounts: map[int]int{6723: 3},
			TemplateIDs:         map[int]int{3: 6726},
			Target:              TargetRandomEnemy,
		},
	}
	primary := &SkillInfo{Template: &skill.SkillTemplate{ID: 6712, Name: "Bách Biến Trùng Kích", SkillType: skill.SkillTypePhysical, AttackCount: 1}, Plan: plan}
	follow := testFollowUpInfo(6726, 2.5)
	processor := NewTurnProcessor(battle, mappedPlanSkillGetter{6712: primary, 6726: follow})
	processor.rng = rand.New(rand.NewSource(11))
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: killed.ID, ActionType: ClientActionSkill, SkillID: 6712})

	followTargets := make(map[string]bool)
	followDamage := make([]int, 0, 5)
	for _, action := range actions {
		if action.ActionType == ActionTypeSkill && action.SkillID == 6726 && action.Damage > 0 {
			followTargets[action.TargetID] = true
			followDamage = append(followDamage, action.Damage)
		}
	}
	if len(followTargets) != 5 {
		t.Fatalf("chained follow-up targets = %v, want all five remaining enemies across two waves", followTargets)
	}
	if alive := battle.GetAliveParticipants(SideEnemy); len(alive) != 0 {
		t.Fatalf("alive enemies after chained waves = %#v, want none", alive)
	}
	if len(followDamage) != 5 {
		t.Fatalf("Trùng Kích damage actions = %v, want five exact-skill hits", followDamage)
	}
	for index, damage := range followDamage {
		if damage < actor.Attack*3/2 {
			t.Fatalf("Trùng Kích hit %d damage = %d, want x2.5 skill damage rather than basic attack damage", index, damage)
		}
	}
	rounds := BuildPlaybackRounds(actions)
	followUpCasts := 0
	for _, round := range rounds {
		for _, action := range round {
			sourceState, _ := action["sSObj"].(map[string]interface{})
			if sourceState["skill"] == "Trùng Kích Truy Kích" {
				followUpCasts++
			}
		}
	}
	if followUpCasts != 2 {
		t.Fatalf("follow-up cast playback count = %d, want two distinct waves; rounds=%#v", followUpCasts, rounds)
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

func TestTurnStartDamagePlaybackUsesAffectedPositionAndUpdatedHP(t *testing.T) {
	battle := NewBattle(BattleTypePVE, 1)
	battle.CurrentRound = 3
	participant := NewParticipant("poisoned", "poisoned", SideEnemy, true)
	participant.Position = 7
	participant.MaxHP = 1000
	participant.CurrentHP = 1000
	participant.IsAlive = true
	battle.AddParticipant(participant)
	processor := NewTurnProcessor(battle, planSkillGetter{})

	actions := processor.applyTurnStartDamage(participant, &TurnStartResult{DamageTaken: 300, PhysicalDamageTaken: 100, MagicDamageTaken: 200})
	if participant.CurrentHP != 700 {
		t.Fatalf("participant HP = %d, want 700", participant.CurrentHP)
	}
	if len(actions) != 2 || actions[0].ActionType != ActionTypeHurt || actions[1].ActionType != ActionTypeIdle {
		t.Fatalf("actions = %#v, want Hurt then Idle", actions)
	}
	dto := actions[0].ToDTO()
	if dto["sid"] != 7 || dto["tid"] != 7 || dto["bid"] != 5 {
		t.Fatalf("periodic damage route = sid %#v tid %#v bid %#v, want 7/7/5", dto["sid"], dto["tid"], dto["bid"])
	}
	sourceState := dto["sSObj"].(map[string]interface{})
	if sourceState["hHp"] != 700 {
		t.Fatalf("sSObj[hHp] = %#v, want 700", sourceState["hHp"])
	}
	if _, exists := dto["tSObj"].(map[string]interface{})["hHp"]; exists {
		t.Fatalf("tSObj unexpectedly contains hHp: %#v", dto["tSObj"])
	}
}

func TestLethalTurnStartDamagePlaybackEndsWithDeath(t *testing.T) {
	battle := NewBattle(BattleTypePVE, 1)
	participant := NewParticipant("poisoned", "poisoned", SideEnemy, true)
	participant.Position = 4
	participant.MaxHP = 100
	participant.CurrentHP = 100
	participant.IsAlive = true
	battle.AddParticipant(participant)
	processor := NewTurnProcessor(battle, planSkillGetter{})

	actions := processor.applyTurnStartDamage(participant, &TurnStartResult{DamageTaken: 150, MagicDamageTaken: 150})
	if participant.IsAlive || participant.CurrentHP != 0 {
		t.Fatalf("participant state = alive %v HP %d, want dead at 0", participant.IsAlive, participant.CurrentHP)
	}
	if len(actions) != 2 || actions[0].ActionType != ActionTypeHurt || actions[1].ActionType != ActionTypeDie {
		t.Fatalf("actions = %#v, want Hurt then Die", actions)
	}
}

func TestAllEnemyPlanDamagesAndDispelsEveryTarget(t *testing.T) {
	battle, actor, first := planTestBattle()
	second := NewParticipant("second", "second", SideEnemy, true)
	second.Position = 2
	second.MaxHP = 1000
	second.CurrentHP = 1000
	third := NewParticipant("third", "third", SideEnemy, true)
	third.Position = 4
	third.MaxHP = 1000
	third.CurrentHP = 1000
	battle.AddParticipant(second)
	battle.AddParticipant(third)
	for _, target := range []*Participant{first, second, third} {
		battle.StatusManager.AddEffect(target.ID, &StatusEffect{Family: "beneficial:" + target.ID, Duration: 2, Polarity: PolarityBeneficial, Removable: true})
	}
	plan := &SkillPlan{Family: "all enemy dispel", Target: TargetAllEnemy, TargetCount: 10, Effects: []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: .8}, {Kind: EffectRemoveOne, Chance: 1}}}
	info := testPlanInfo(plan)
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: first.ID, ActionType: ClientActionSkill, SkillID: info.Template.ID})
	for _, target := range []*Participant{first, second, third} {
		if target.CurrentHP >= target.MaxHP {
			t.Fatalf("target %s was not damaged", target.ID)
		}
		if len(battle.StatusManager.GetEffects(target.ID)) != 0 {
			t.Fatalf("target %s buff was not removed", target.ID)
		}
	}
	assertDamagePopupsForTargets(t, BuildPlaybackRounds(actions), first, second, third)
}

func TestFixedAOEDamageAlwaysProducesDamagePopupBeforeDeath(t *testing.T) {
	battle, actor, first := planTestBattle()
	second := NewParticipant("second", "second", SideEnemy, true)
	second.Position = 2
	second.MaxHP = 1000
	second.CurrentHP = 1000
	battle.AddParticipant(second)
	first.CurrentHP = 100
	second.CurrentHP = 100
	plan := &SkillPlan{Family: "lethal aoe", Target: TargetAllEnemy, TargetCount: 10, Effects: []SkillEffect{{Kind: EffectFixedMaxHPDamage, Coefficient: .5}}}
	info := testPlanInfo(plan)
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: first.ID, ActionType: ClientActionSkill, SkillID: info.Template.ID})
	if first.IsAlive || second.IsAlive {
		t.Fatalf("lethal AoE left targets alive: first=%v second=%v", first.IsAlive, second.IsAlive)
	}
	assertDamagePopupsForTargets(t, BuildPlaybackRounds(actions), first, second)
}

func TestRemoveAllEffectsRemovesOnlyEligiblePolarity(t *testing.T) {
	manager := NewStatusEffectManager()
	participantID := "target"
	manager.AddEffect(participantID, &StatusEffect{Family: "good-1", Polarity: PolarityBeneficial, Removable: true})
	manager.AddEffect(participantID, &StatusEffect{Family: "good-locked", Polarity: PolarityBeneficial, Removable: false})
	manager.AddEffect(participantID, &StatusEffect{Family: "bad", Polarity: PolarityHarmful, Removable: true})
	manager.AddEffect(participantID, &StatusEffect{Family: "good-2", Polarity: PolarityBeneficial, Removable: true})
	if removed := manager.RemoveAllEffects(participantID, PolarityBeneficial); removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	effects := manager.GetEffects(participantID)
	if len(effects) != 2 || effects[0].Family != "good-locked" || effects[1].Family != "bad" {
		t.Fatalf("remaining effects = %#v", effects)
	}
}

func TestRandomEnemyPlanUsesConfiguredTargetCount(t *testing.T) {
	battle, actor, _ := planTestBattle()
	for i := 1; i < 10; i++ {
		target := NewParticipant("enemy-"+strconv.Itoa(i), "enemy", SideEnemy, true)
		target.Position = i
		target.MaxHP = 1000
		target.CurrentHP = 1000
		battle.AddParticipant(target)
	}
	resolver := NewTargetResolverWithRNG(battle, rand.New(rand.NewSource(11)))
	targets := resolver.ResolveTargets(actor, "", TargetRandomEnemy, AreaNone, 6)
	if len(targets) != 6 {
		t.Fatalf("target count = %d, want 6", len(targets))
	}
	seen := make(map[string]bool)
	for _, target := range targets {
		if target.Side != SideEnemy || seen[target.ID] {
			t.Fatalf("invalid random targets = %#v", targets)
		}
		seen[target.ID] = true
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

func testFollowUpInfo(id int, coefficient float64) *SkillInfo {
	return &SkillInfo{
		Template: &skill.SkillTemplate{ID: id, Name: "Trùng Kích Truy Kích", SkillType: skill.SkillTypePhysical, AttackCount: 1, Range: 1},
		Plan:     &SkillPlan{Family: "Trùng Kích Truy Kích", Target: TargetEnemy, TargetCount: 1, Effects: []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: coefficient}}},
	}
}

func assertDamagePopupsForTargets(t *testing.T, rounds [][]map[string]interface{}, targets ...*Participant) {
	t.Helper()
	want := make(map[int]bool, len(targets))
	for _, target := range targets {
		want[target.Position] = true
	}
	found := make(map[int]bool, len(targets))
	for _, round := range rounds {
		for _, action := range round {
			if state, ok := action["tSObj"].(map[string]interface{}); ok {
				if _, ok := state["hHp"]; ok {
					if position, ok := action["tid"].(int); ok && want[position] {
						found[position] = true
					}
				}
			}
			if state, ok := action["sSObj"].(map[string]interface{}); ok {
				if _, ok := state["hHp"]; ok {
					if position, ok := action["sid"].(int); ok && want[position] {
						found[position] = true
					}
				}
			}
		}
	}
	if len(found) != len(want) {
		t.Fatalf("damage popups found at positions %v, want %v; rounds=%#v", found, want, rounds)
	}
}
