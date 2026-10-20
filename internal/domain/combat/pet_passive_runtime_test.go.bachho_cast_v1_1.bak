package combat

import "testing"

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
