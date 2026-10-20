package combat

import (
	"math/rand"
	"mcgame-server/internal/domain/skill"
	"testing"
)

func TestBachHoTrueDamageIgnoresArmor(t *testing.T) {
	p := NewParticipant("tiger", "tiger", SidePlayer, false)
	p.Attack = 210
	e := SkillEffect{TrueDamageChance: 1, TrueDamageAttackRatio: 1}
	if got := petTrueDamageBonus(p, e, .2); got != 210 {
		t.Fatalf("true damage=%d want 210", got)
	}
	if got := petTrueDamageBonus(p, SkillEffect{TrueDamageChance: .2, TrueDamageAttackRatio: 1}, .5); got != 0 {
		t.Fatalf("proc failed: %d", got)
	}
}

func TestBachHoBloodLevelsAndAttack(t *testing.T) {
	b, p, _ := planTestBattle()
	tp := NewTurnProcessor(b, nil)
	p.SkillIDs[7030] = true
	if action := tp.activatePetTurnBuff(p); action == nil || tp.statusManager.petBattleBuffRank(p.ID) != 2 {
		t.Fatal("level-2 blood buff missing")
	}
	p.MaxHP = 1000
	p.CurrentHP = 500
	p.Attack = 200
	// At half HP, level 2 gives +40% Attack.
	if got := tp.statusManager.EffectiveParticipant(p).Attack; got != 280 {
		t.Fatalf("attack %d want 280", got)
	}
	if action := tp.activatePetTurnBuff(p); action != nil {
		t.Fatal("re-applied buff icon twice")
	}
	tp.statusManager.RemoveFamily(p.ID, "PetPassive:ChienDauMauLua")
	if got := tp.statusManager.EffectiveParticipant(p).Attack; got != 200 {
		t.Fatalf("buff removal: attack %d", got)
	}
}

func TestBachHoReflectAndSecretStrike(t *testing.T) {
	b, actor, tiger := planTestBattle()
	actor.Side = SideEnemy
	tiger.Side = SidePlayer
	tiger.SkillIDs[7031] = true
	tp := NewTurnProcessor(b, nil)
	if tp.activatePetTurnBuff(tiger) == nil {
		t.Fatal("no buff add action")
	}
	actor.MaxHP = 2000
	actor.CurrentHP = 2000
	before := actor.CurrentHP
	actions := tp.petReflectActions(actor, tiger, 100)
	if len(actions) == 0 || before-actor.CurrentHP != 50 {
		t.Fatalf("reflect=%d actions=%d", before-actor.CurrentHP, len(actions))
	}

	// Separately verify the passive levels and exact one-cast policy.
	if ch, _ := petSecretStrikeRank(map[int]bool{7034: true}); ch != .80 {
		t.Fatalf("secret chance %v", ch)
	}
	b2, pet, killed := planTestBattle()
	killed.CurrentHP = 0
	killed.IsAlive = false
	pet.SkillIDs[7034] = true
	pet.Attack = 500
	for i := 0; i < 4; i++ {
		e := NewParticipant(string(rune('a'+i)), "foe", SideEnemy, true)
		e.Position = i + 1
		e.MaxHP = 10000
		e.CurrentHP = 10000
		e.Defense = 50
		b2.AddParticipant(e)
	}
	tp2 := NewTurnProcessor(b2, nil)
	tp2.rng = rand.New(rand.NewSource(1))
	tpl := &skill.SkillTemplate{ID: 7017, Name: "Thú Vương Đột Kích", SkillType: skill.SkillTypePhysical, AttackCount: 1}
	results := tp2.executePetOnKillStrike(pet, tpl)
	if len(results) == 0 {
		t.Fatal("secret strike did not trigger on seeded RNG")
	}
	damaged := 0
	for _, target := range b2.GetAliveParticipants(SideEnemy) {
		if target.CurrentHP < 10000 {
			damaged++
		}
	}
	if damaged != 3 {
		t.Fatalf("secret strike hit %d targets; want 3", damaged)
	}
}
