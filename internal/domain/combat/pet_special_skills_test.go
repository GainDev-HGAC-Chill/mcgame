// Special Pet skill tests cover composite damage, multi-row targeting, effective stats, and buff icon lifecycle.
package combat

import (
	"math"
	"math/rand"
	"strconv"
	"testing"

	"mcgame-server/internal/domain/skill"
	"mcgame-server/internal/domain/stats"
)

func TestUMinhHuyenVuTargetsEightAcrossEnemyRows(t *testing.T) {
	battle, actor, selected := planTestBattle()
	for position := 1; position < 10; position++ {
		target := NewParticipant("u-minh-"+strconv.Itoa(position), "enemy", SideEnemy, true)
		target.Position = position
		target.MaxHP = 1000
		target.CurrentHP = 1000
		battle.AddParticipant(target)
	}
	targets := NewTargetResolverWithRNG(battle, rand.New(rand.NewSource(7))).ResolveTargets(actor, selected.ID, TargetEnemy, AreaHorizontalRows, 8)
	if len(targets) != 8 {
		t.Fatalf("U Minh targets = %d, want 8 across both enemy rows", len(targets))
	}
	seen := make(map[int]bool, len(targets))
	for _, target := range targets {
		if target.Side != SideEnemy || seen[target.Position] {
			t.Fatalf("invalid U Minh targets = %#v", targets)
		}
		seen[target.Position] = true
	}
	if !seen[selected.Position] {
		t.Fatalf("selected target position %d missing from %v", selected.Position, seen)
	}
}

func TestUMinhHuyenVuDamageIncludesAttackAndCasterMaxHP(t *testing.T) {
	battle, actor, target := planTestBattle()
	actor.Attack = 100
	actor.MaxHP = 1000
	target.MaxHP = 10000
	target.CurrentHP = 10000
	target.Defense = 0
	processor := NewTurnProcessor(battle, nil)
	tpl := &skill.SkillTemplate{ID: 7008, Name: "U Minh Huyền Vũ", SkillType: skill.SkillTypePhysical, AttackCount: 1, Range: 2}
	actions := processor.applyPlannedDamage(actor, target, 7008, tpl, SkillEffect{Kind: EffectPhysicalDamage, Coefficient: .25, CasterMaxHPRatio: .25})
	if len(actions) == 0 || actions[0].Damage < 200 {
		t.Fatalf("U Minh composite damage = %#v, want 25%% Attack + 25%% MaxHP before defenses", actions)
	}
	v2Template := *tpl
	v2Template.Multiplier = .25
	v2Template.BaseDamage = 250
	hits := ResolveHits(actor, target, &SkillInfo{Template: &v2Template}, rand.New(rand.NewSource(5)))
	if len(hits) != 1 || hits[0].Damage != 275 {
		t.Fatalf("per-mille U Minh composite damage = %#v, want 275", hits)
	}
}

func TestThatHonBaseChanceSubtractsGenericAndStunResistance(t *testing.T) {
	battle, actor, target := planTestBattle()
	target.EffStats.Set(stats.PropResistDebuff, 100)
	target.EffStats.Set(stats.PropResistStun, 200)
	processor := NewTurnProcessor(battle, nil)
	effect := SkillEffect{Kind: EffectStatus, Chance: .60, Polarity: PolarityHarmful, Control: ControlEffect{Stun: true}}
	if chance := processor.plannedEffectChance(actor, target, effect); math.Abs(chance-.30) > .000001 {
		t.Fatalf("Thất Hồn chance = %.3f, want 0.300 after 100 generic + 200 stun resistance", chance)
	}
}

func TestThoSanSatThuAppliesPercentStatsAndBuffIcon(t *testing.T) {
	battle, actor, ally := planTestBattle()
	ally.Side = actor.Side
	ally.Speed = 100
	ally.Hit = 100
	plan := &SkillPlan{Family: "Thợ Săn Sát Thủ", Target: TargetAllTeam, TargetCount: 10, Effects: []SkillEffect{{Kind: EffectStatus, Family: "Thợ Săn Sát Thủ", Duration: 5, Polarity: PolarityBeneficial, Removable: true, BuffID: 80118, Modifiers: []StatModifier{{Stat: StatSpeed, Percent: .1}, {Stat: StatAccuracy, Percent: .1}}}}}
	info := &SkillInfo{Template: &skill.SkillTemplate{ID: 7014, Name: "Thợ Săn Sát Thủ", SkillType: skill.SkillTypeBuff, AttackCount: 1}, Plan: plan}
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, ActionType: ClientActionSkill, SkillID: 7014})
	effective := battle.StatusManager.EffectiveParticipant(ally)
	if effective.Speed != 110 || effective.Hit != 110 {
		t.Fatalf("Thợ Săn effective speed/hit = %d/%d, want 110/110", effective.Speed, effective.Hit)
	}
	assertBuffAddPlayback(t, BuildPlaybackRounds(actions), ally.Position, 80118)
}

func TestTuPhuongKhiepDamAppliesDebuffAndBuffIcon(t *testing.T) {
	battle, actor, target := planTestBattle()
	target.Attack = 100
	target.MagicAttack = 100
	target.Speed = 100
	plan := &SkillPlan{Family: "Tứ Phương Khiếp Đảm", Target: TargetRandomEnemy, TargetCount: 1, Effects: []SkillEffect{{Kind: EffectStatus, Family: "Tứ Phương Khiếp Đảm", Duration: 1, Polarity: PolarityHarmful, Removable: true, BuffID: 80121, Modifiers: []StatModifier{{Stat: StatAttack, Percent: -.05}, {Stat: StatMagicAttack, Percent: -.05}, {Stat: StatSpeed, Percent: -.05}}}}}
	info := &SkillInfo{Template: &skill.SkillTemplate{ID: 7011, Name: "Tứ Phương Khiếp Đảm", SkillType: skill.SkillTypeDebuff, AttackCount: 1}, Plan: plan}
	processor := NewTurnProcessor(battle, planSkillGetter{info: info})
	processor.rng = rand.New(rand.NewSource(7))
	actions := processor.executeSkill(actor, &BattleCommand{ActorID: actor.ID, TargetID: target.ID, ActionType: ClientActionSkill, SkillID: 7011})
	effective := battle.StatusManager.EffectiveParticipant(target)
	if effective.Attack != 95 || effective.MagicAttack != 95 || effective.Speed != 95 {
		t.Fatalf("Tứ Phương effective stats = %d/%d/%d, want 95/95/95", effective.Attack, effective.MagicAttack, effective.Speed)
	}
	assertBuffAddPlayback(t, BuildPlaybackRounds(actions), target.Position, 80121)
	result := &RoundResult{}
	processor.appendTurnEndStatusActions(result, target)
	if len(result.Actions) != 1 {
		t.Fatalf("turn-end status actions = %#v, want one buff removal", result.Actions)
	}
	removed, ok := result.Actions[0].SourceState["buffColDel"].([]int)
	if !ok || len(removed) != 1 || removed[0] != 80121 {
		t.Fatalf("turn-end buffColDel = %#v, want [80121]", result.Actions[0].SourceState["buffColDel"])
	}
}

func assertBuffAddPlayback(t *testing.T, rounds [][]map[string]interface{}, targetPosition, buffID int) {
	t.Helper()
	for _, round := range rounds {
		for _, action := range round {
			if action["tid"] != targetPosition {
				continue
			}
			targetState, _ := action["tSObj"].(map[string]interface{})
			add, ok := targetState["buffAdd"].(map[string]interface{})
			if ok && add["id"] == buffID {
				return
			}
		}
	}
	t.Fatalf("missing buffAdd %d for target position %d in %#v", buffID, targetPosition, rounds)
}
