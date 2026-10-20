package combat

import (
	"math/rand"
	"testing"

	"mcgame-server/internal/domain/skill"
)

func TestPetPassiveRulesDataAndTriggers(t *testing.T) {
	book, err := parsePetPassiveRules(petPassiveRulesJSON)
	if err != nil {
		t.Fatalf("invalid embedded passive rules: %v", err)
	}
	if len(book.Rules) != 3 {
		t.Fatalf("expected 3 migrated rule families, got %d", len(book.Rules))
	}
	if got := petTurnBuffRank(map[int]bool{7029: true, 7031: true}); got != 3 {
		t.Fatalf("highest turn buff rank: %d", got)
	}
	if !petHasOnKillProc(7017) || !petHasOnKillProc(7025) || petHasOnKillProc(7038) {
		t.Fatal("on-kill trigger list not scoped")
	}
	if chance, rank := petSecretStrikeRank(map[int]bool{7034: true}); chance != .8 || rank != 3 {
		t.Fatalf("on-kill passive level got chance=%v rank=%d", chance, rank)
	}
	if effect, ok := petLinkedControlEffect(7038, map[int]bool{7048: true}); !ok || effect.BuffID != 893 || effect.Duration != 2 || effect.Chance != 1.30 {
		t.Fatalf("linked control: %+v, enabled=%v", effect, ok)
	}
	if _, ok := petLinkedControlEffect(7017, map[int]bool{7048: true}); ok {
		t.Fatal("unrelated trigger enabled linked control")
	}
}

func TestPetPassiveRulesRejectDuplicatesAndBadInput(t *testing.T) {
	if _, err := parsePetPassiveRules([]byte(`{"version":2,"rules":[]}`)); err == nil {
		t.Fatal("accepted wrong version")
	}
	if _, err := parsePetPassiveRules([]byte(`{"version":1,"rules":[{"kind":"unknown","family":"X","levels":[{"skill_id":123}]}]}`)); err == nil {
		t.Fatal("accepted unknown rule")
	}
}

func TestPetPassiveRulesAllowSeveralPetsPerTriggerType(t *testing.T) {
	raw := []byte(`{"version":1,"rules":[
      {"kind":"on_hit_control","family":"pet1","trigger_skill_ids":[100],"buff_id":89,"duration":2,"levels":[{"skill_id":101,"chance":0.5}]},
      {"kind":"on_hit_control","family":"pet2","trigger_skill_ids":[200],"buff_id":90,"duration":2,"levels":[{"skill_id":201,"chance":0.6}]}
    ]}`)
	book, err := parsePetPassiveRules(raw)
	if err != nil || len(book.Rules) != 2 {
		t.Fatalf("multiple pet families should be accepted: %v", err)
	}
	// Duplicate trigger IDs in a single event type are ambiguous and rejected.
	raw = []byte(`{"version":1,"rules":[
      {"kind":"on_hit_control","family":"pet1","trigger_skill_ids":[100],"buff_id":89,"duration":2,"levels":[{"skill_id":101,"chance":0.5}]},
      {"kind":"on_hit_control","family":"pet2","trigger_skill_ids":[100],"buff_id":90,"duration":2,"levels":[{"skill_id":201,"chance":0.6}]}
    ]}`)
	if _, err := parsePetPassiveRules(raw); err == nil {
		t.Fatal("duplicate control trigger should fail")
	}
}

// Regression: the follow-up is a DIFFERENT skill cast, not a 4-target primary hit.
func TestPetSecretStrikeSeparatePlaybackSkillAndThreeDistinctTargets(t *testing.T) {
	battle, pet, original := planTestBattle()
	original.IsAlive = false
	original.CurrentHP = 0
	pet.SkillIDs[7033] = true // 60% middle rank
	pet.Attack = 500
	for i := 0; i < 4; i++ {
		foe := NewParticipant(string(rune('a'+i)), "enemy", SideEnemy, true)
		foe.Position = 1 + i
		foe.MaxHP, foe.CurrentHP, foe.Defense = 50000, 50000, 0
		battle.AddParticipant(foe)
	}
	tp := NewTurnProcessor(battle, nil)
	tp.rng = rand.New(rand.NewSource(2)) // first 60% roll succeeds
	primary := &skill.SkillTemplate{ID: 7018, Name: "Thú Vương Đột Kích", SkillType: skill.SkillTypePhysical, AttackCount: 1}
	actions := tp.executePetOnKillStrike(pet, primary)
	hits := map[string]bool{}
	for _, action := range actions {
		if action.ActionType != ActionTypeSkill {
			continue
		}
		if action.SkillID != 7033 || action.SkillName != "Đột Kích Bí Mật" {
			t.Fatalf("follow-up cast replayed primary skill: id=%d name=%q", action.SkillID, action.SkillName)
		}
		if action.TargetID == original.ID {
			t.Fatal("dead original included in secret targets")
		}
		hits[action.TargetID] = true
	}
	if len(hits) != 3 {
		t.Fatalf("secondary cast hit %d distinct targets, want 3", len(hits))
	}
	// The secondary skill has its own visible name in the Flash battle DTO.
	dto := BuildPlaybackRounds(actions)
	visible := false
	for _, round := range dto {
		for _, frame := range round {
			state, ok := frame["sSObj"].(map[string]interface{})
			if ok && state["skill"] == "Đột Kích Bí Mật" {
				visible = true
			}
		}
	}
	if !visible {
		t.Fatal("secondary skill name missing from playback")
	}
}

func TestPetSecretStrikeMustOwnPassive(t *testing.T) {
	battle, pet, original := planTestBattle()
	original.IsAlive = false
	pet.Attack = 500
	foe := NewParticipant("extra", "extra", SideEnemy, true)
	foe.Position = 2
	foe.MaxHP, foe.CurrentHP = 50000, 50000
	battle.AddParticipant(foe)
	tp := NewTurnProcessor(battle, nil)
	primary := &skill.SkillTemplate{ID: 7018, SkillType: skill.SkillTypePhysical, AttackCount: 1}
	if actions := tp.executePetOnKillStrike(pet, primary); len(actions) != 0 {
		t.Fatal("secondary hit without owned passive")
	}
}

func TestPetReflectionScalesWithActualReceivedDamage(t *testing.T) {
	battle, pet, attacker := planTestBattle()
	pet.SkillIDs[7030] = true // 30% incoming-damage reflection
	pet.MaxHP, pet.CurrentHP = 10000, 10000
	attacker.MaxHP, attacker.CurrentHP = 10000, 10000
	attacker.Defense = 100000 // reflect is not a normal physical attack against armor
	tp := NewTurnProcessor(battle, nil)
	if action := tp.activatePetTurnBuff(pet); action == nil {
		t.Fatal("reflect buff not activated")
	}
	before := attacker.CurrentHP
	hits := tp.petReflectActions(attacker, pet, 1200)
	if before-attacker.CurrentHP != 360 || len(hits) == 0 {
		t.Fatalf("incoming 1200, rank2 30%% should return 360 damage, got %d", before-attacker.CurrentHP)
	}
	// No feedback loop: reflect actions are playback, not another direct-attack execution.
	if attacker.CurrentHP != 9640 {
		t.Fatalf("reflected damage loop: hp=%d", attacker.CurrentHP)
	}
}
