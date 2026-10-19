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
