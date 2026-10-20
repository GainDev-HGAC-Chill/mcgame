package combat

import (
	"math/rand"
	"testing"
)

func TestLoiLanAnhPassiveRanksAndScope(t *testing.T) {
	for _, tc := range []struct {
		rank   int
		chance float64
	}{{7047, 1.10}, {7048, 1.30}, {7049, 1.50}} {
		effect, ok := petLinkedControlEffect(7038, map[int]bool{tc.rank: true})
		if !ok || effect.Chance != tc.chance || effect.BuffID != 893 || effect.Duration != 2 || effect.Control.TauntSourceID != "caster" {
			t.Fatalf("passive %d: %#v (ok=%v)", tc.rank, effect, ok)
		}
	}
	if _, ok := petLinkedControlEffect(7002, map[int]bool{7049: true}); ok {
		t.Fatal("Sức Mạnh Phán Quyết triggered from another pet skill")
	}
	if _, ok := petLinkedControlEffect(7038, nil); ok {
		t.Fatal("passive triggered without being learned")
	}
}

func TestLoiLanAnhTauntIconDurationAndForcedAttack(t *testing.T) {
	battle, pet, enemy := planTestBattle()
	pet.MaxHP = 3000
	pet.CurrentHP = pet.MaxHP
	pet.SkillIDs[7047] = true
	pet.Hit = 500
	enemy.MaxHP = 100000
	enemy.CurrentHP = enemy.MaxHP
	info := testPlanInfo(&SkillPlan{
		Family: "Lôi Đình Thần Phạt", Target: TargetEnemy, TargetCount: 1,
		Effects: []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: 0.5}},
	})
	info.Template.ID = 7038
	tp := NewTurnProcessor(battle, planSkillGetter{info: info})
	tp.rng = rand.New(rand.NewSource(23))
	actions := tp.executeSkillPlan(pet, &BattleCommand{ActorID: pet.ID, TargetID: enemy.ID, SkillID: 7038}, info)
	icon := false
	for _, action := range actions {
		if payload, ok := action.TargetState["buffAdd"].(map[string]interface{}); ok && payload["id"] == 893 {
			icon = true
		}
	}
	if !icon || tp.statusManager.TauntTarget(enemy.ID) != pet.ID {
		t.Fatalf("Trào Phúng not applied: icon=%v taunt=%s", icon, tp.statusManager.TauntTarget(enemy.ID))
	}
	attack := tp.executeCommand(enemy, &BattleCommand{ActorID: enemy.ID, TargetID: enemy.ID, ActionType: ClientActionSkill, SkillID: 12345})
	forcedBasicAttack := false
	for _, action := range attack {
		if action.ActionType == ActionTypeAttack && action.TargetID == pet.ID {
			forcedBasicAttack = true
		}
	}
	if !forcedBasicAttack {
		t.Fatalf("taunted enemy did not use basic attack against pet: %#v", attack)
	}
	if removed := tp.statusManager.ProcessTurnEnd(enemy.ID); len(removed) != 0 {
		t.Fatalf("taunt expired before 2 turns: %v", removed)
	}
	if removed := tp.statusManager.ProcessTurnEnd(enemy.ID); len(removed) != 1 || removed[0] != 893 {
		t.Fatalf("missing buff 893 removal after 2 turns: %v", removed)
	}
	if tp.statusManager.TauntTarget(enemy.ID) != "" {
		t.Fatal("Trào Phúng remained after expiration")
	}
}

func TestLoiLanAnhBonusUsesCasterMaxHP(t *testing.T) {
	makeDamage := func(bonus bool) int {
		battle, pet, enemy := planTestBattle()
		pet.MaxHP = 6000
		pet.CurrentHP = 6000
		pet.Hit = 1000
		enemy.MaxHP = 100000
		enemy.CurrentHP = 100000
		effect := SkillEffect{Kind: EffectPhysicalDamage, Coefficient: 0.5}
		if bonus {
			effect.BonusCasterMaxHPChance = 1
			effect.BonusCasterMaxHPRatio = .10
		}
		info := testPlanInfo(&SkillPlan{Family: "test", Target: TargetEnemy, TargetCount: 1, Effects: []SkillEffect{effect}})
		tp := NewTurnProcessor(battle, planSkillGetter{info: info})
		tp.rng = rand.New(rand.NewSource(99))
		actions := tp.executeSkillPlan(pet, &BattleCommand{ActorID: pet.ID, TargetID: enemy.ID, SkillID: info.Template.ID}, info)
		if len(actions) == 0 {
			t.Fatal("missing damage action")
		}
		return enemy.MaxHP - enemy.CurrentHP
	}
	normal := makeDamage(false)
	extra := makeDamage(true)
	if extra-normal < 450 || extra-normal > 750 {
		t.Fatalf("caster max-HP bonus = %d (normal %d, proc %d), want close to 600", extra-normal, normal, extra)
	}
}

func TestLoiLanAnhExtraCastPaysOnlyOnce(t *testing.T) {
	battle, pet, enemy := planTestBattle()
	enemy.MaxHP = 100000
	enemy.CurrentHP = 100000
	pet.Hit = 1000
	info := testPlanInfo(&SkillPlan{
		Family: "Lôi Đình Thần Phạt", Target: TargetEnemy, TargetCount: 1,
		RepeatChance: 1,
		Effects:      []SkillEffect{{Kind: EffectPhysicalDamage, Coefficient: .25}},
	})
	info.Template.MPCost = 13
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	processor.rng = rand.New(rand.NewSource(11))
	actions := processor.executeSkillPlan(pet, &BattleCommand{
		ActorID: pet.ID, TargetID: enemy.ID, SkillID: info.Template.ID,
	}, info)
	strikes := 0
	for _, action := range actions {
		if action.ActionType == ActionTypeSkill && action.Damage > 0 && action.TargetID == enemy.ID {
			strikes++
		}
	}
	if strikes != 2 || pet.CurrentMP != 1000-13 {
		t.Fatalf("extra cast strikes=%d remainingMP=%d, want 2 strikes, MP 987", strikes, pet.CurrentMP)
	}
}
