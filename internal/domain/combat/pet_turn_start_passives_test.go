// Pet turn-start passive tests verify exact cleanse chances and effective debuff resistance.
package combat

import (
	"math"
	"math/rand"
	"testing"

	"mcgame-server/internal/domain/stats"
)

func TestDiHinhHoanViCatalogLevels(t *testing.T) {
	want := map[int]diHinhHoanViLevel{
		6751: {chance: 0.20, resistance: 80},
		6752: {chance: 0.30, resistance: 120},
		6753: {chance: 0.40, resistance: 150},
	}
	for skillID, expected := range want {
		if got := diHinhHoanViLevels[skillID]; got != expected {
			t.Fatalf("skill %d config = %#v, want %#v", skillID, got, expected)
		}
	}
}

func TestDiHinhHoanViCleansesDebuffsAndAddsRealResistance(t *testing.T) {
	battle, actor, caster := planTestBattle()
	actor.SkillIDs[6753] = true
	manager := battle.StatusManager
	manager.AddEffect(actor.ID, &StatusEffect{Family: "poison", Duration: 3, Polarity: PolarityHarmful, Removable: true, Power: 100, Periodic: &PeriodicEffect{Damage: true, Coefficient: 1}})
	manager.AddEffect(actor.ID, &StatusEffect{Family: "locked", Duration: 3, Polarity: PolarityHarmful, Removable: false})
	manager.AddEffect(actor.ID, &StatusEffect{Family: "beneficial", Duration: 3, Polarity: PolarityBeneficial, Removable: true})
	processor := NewTurnProcessor(battle, nil)
	processor.rng = rand.New(rand.NewSource(2))

	if !processor.activateDiHinhHoanVi(actor) {
		t.Fatal("level-3 Di Hình Hoán Vị did not activate on deterministic successful roll")
	}
	if result := manager.ProcessTurnStart(actor.ID, actor); result.DamageTaken != 0 {
		t.Fatalf("cleansed periodic debuff still dealt %d damage", result.DamageTaken)
	}
	effects := manager.SortedEffects(actor.ID)
	if len(effects) != 3 || effects[0].Family != diHinhHoanViResistanceFamily || effects[1].Family != "beneficial" || effects[2].Family != "locked" {
		t.Fatalf("effects after cleanse = %#v", effects)
	}
	if got := manager.EffectiveParticipant(actor).EffStats.Get(stats.PropResistDebuff); got != 150 {
		t.Fatalf("effective debuff resistance = %d, want 150", got)
	}
	caster.EffStats.Set(stats.PropDebuffHit, 50)
	chance := processor.plannedEffectChance(caster, actor, SkillEffect{Kind: EffectStatus, Chance: 0.4, Polarity: PolarityHarmful})
	if math.Abs(chance-0.3) > 0.000001 {
		t.Fatalf("harmful status chance = %.3f, want 0.300 after 50 hit - 150 resistance", chance)
	}
	guaranteedChance := processor.plannedEffectChance(caster, actor, SkillEffect{Kind: EffectStatus, Polarity: PolarityHarmful})
	if math.Abs(guaranteedChance-0.9) > 0.000001 {
		t.Fatalf("base-guaranteed harmful status chance = %.3f, want 0.900 after 50 hit - 150 resistance", guaranteedChance)
	}
}

func TestDiHinhHoanViFailedRollExpiresPreviousResistance(t *testing.T) {
	battle, actor, _ := planTestBattle()
	actor.SkillIDs[6753] = true
	battle.StatusManager.AddEffect(actor.ID, &StatusEffect{Duration: 1, Family: diHinhHoanViResistanceFamily, Polarity: PolarityBeneficial, Modifiers: []StatModifier{{Stat: StatDebuffResistance, Flat: 150}}})
	processor := NewTurnProcessor(battle, nil)
	processor.rng = rand.New(rand.NewSource(1))

	if processor.activateDiHinhHoanVi(actor) {
		t.Fatal("level-3 Di Hình Hoán Vị unexpectedly activated on deterministic failed roll")
	}
	if got := battle.StatusManager.EffectiveParticipant(actor).EffStats.Get(stats.PropResistDebuff); got != 0 {
		t.Fatalf("expired debuff resistance = %d, want 0", got)
	}
}

func TestDaKhangTinhCatalogAndGuaranteedCleanse(t *testing.T) {
	want := map[int]float64{3808: .50, 3812: .75, 3813: 1}
	for skillID, chance := range want {
		if got := daKhangTinhLevels[skillID]; got != chance {
			t.Fatalf("skill %d chance = %.2f, want %.2f", skillID, got, chance)
		}
	}
	battle, actor, _ := planTestBattle()
	actor.SkillIDs[3813] = true
	battle.StatusManager.AddEffect(actor.ID, &StatusEffect{Family: "poison", BuffID: 936, Duration: 3, Polarity: PolarityHarmful, Removable: true, Periodic: &PeriodicEffect{Damage: true, Fixed: 25}})
	battle.StatusManager.AddEffect(actor.ID, &StatusEffect{Family: "locked", BuffID: 999, Duration: 3, Polarity: PolarityHarmful, Removable: false})
	processor := NewTurnProcessor(battle, nil)
	removed, buffID := processor.activateDaKhangTinh(actor)
	if !removed || buffID != 936 {
		t.Fatalf("cleanse = %v/%d, want true/936", removed, buffID)
	}
	if result := battle.StatusManager.ProcessTurnStart(actor.ID, actor); result.DamageTaken != 0 {
		t.Fatalf("cleansed debuff dealt %d damage", result.DamageTaken)
	}
	if effects := battle.StatusManager.GetEffects(actor.ID); len(effects) != 1 || effects[0].Family != "locked" {
		t.Fatalf("remaining effects = %#v", effects)
	}
}

func TestDaKhangTinhFailedRollLeavesDebuff(t *testing.T) {
	battle, actor, _ := planTestBattle()
	actor.SkillIDs[3808] = true
	battle.StatusManager.AddEffect(actor.ID, &StatusEffect{Family: "slow", BuffID: 497, Duration: 3, Polarity: PolarityHarmful, Removable: true})
	processor := NewTurnProcessor(battle, nil)
	processor.rng = rand.New(rand.NewSource(1))
	if removed, _ := processor.activateDaKhangTinh(actor); removed {
		t.Fatal("level-1 Da Kháng Tính unexpectedly cleansed on failed roll")
	}
	if effects := battle.StatusManager.GetEffects(actor.ID); len(effects) != 1 {
		t.Fatalf("effects after failed roll = %#v", effects)
	}
}

func TestCanVeThanhNuCreatesAndRefreshesShieldWithIcon(t *testing.T) {
	want := map[int]canVeThanhNuLevel{
		3848: {ratio: .50, buffID: 2015},
		3849: {ratio: .75, buffID: 2014},
		3850: {ratio: 1, buffID: 2013},
	}
	for skillID, expected := range want {
		if got := canVeThanhNuLevels[skillID]; got != expected {
			t.Fatalf("skill %d level = %#v, want %#v", skillID, got, expected)
		}
	}
	battle, actor, _ := planTestBattle()
	actor.MaxMP = 200
	actor.SkillIDs[3848] = true
	processor := NewTurnProcessor(battle, nil)
	actions := processor.activatePetTurnStartPassives(actor)
	if len(actions) != 1 {
		t.Fatalf("turn-start actions = %#v", actions)
	}
	add, ok := actions[0].SourceState["buffAdd"].(map[string]interface{})
	if !ok || add["id"] != 2015 {
		t.Fatalf("buffAdd = %#v, want 2015", actions[0].SourceState["buffAdd"])
	}
	effects := battle.StatusManager.GetEffects(actor.ID)
	if len(effects) != 1 || effects[0].Shield == nil || effects[0].Shield.Pool != 100 || effects[0].Shield.Hits != 3 || effects[0].BuffID != 2015 {
		t.Fatalf("shield effect = %#v", effects)
	}
	if damage := battle.StatusManager.ApplyDamage(actor.ID, 60, false); damage != 0 {
		t.Fatalf("shielded damage = %d, want 0", damage)
	}
	if buffID := processor.activateCanVeThanhNu(actor); buffID != 2015 {
		t.Fatalf("refresh buff ID = %d, want 2015", buffID)
	}
	effects = battle.StatusManager.GetEffects(actor.ID)
	if len(effects) != 1 || effects[0].Shield.Pool != 100 || effects[0].Shield.Hits != 3 {
		t.Fatalf("refreshed shield = %#v", effects)
	}
}
