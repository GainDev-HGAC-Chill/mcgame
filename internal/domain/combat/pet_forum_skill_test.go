// Forum skill runtime tests cover planned petrification lifecycle.
package combat

import "testing"

func TestPlannedStoneBlocksActionsAndUsesBuffID(t *testing.T) {
	battle, actor, target := planTestBattle()
	processor := NewTurnProcessor(battle, nil)
	effect := SkillEffect{
		Kind: EffectStatus, Chance: 1, Duration: 2, Polarity: PolarityHarmful,
		Removable: true, Family: "Thạch Hóa Công Kích", BuffID: 299,
		Control: ControlEffect{Stone: true},
	}
	processor.addPlannedStatus(actor, target, 2615, effect)
	effects := processor.statusManager.GetEffects(target.ID)
	if len(effects) != 1 || effects[0].Type != StatusStone || effects[0].BuffID != 299 {
		t.Fatalf("stone effects = %#v", effects)
	}
	if processor.statusManager.CanAct(target.ID) || processor.statusManager.CanUseSkill(target.ID) {
		t.Fatal("stone target can still act or use skills")
	}
	removed, buffID := processor.statusManager.RemoveRandomEffectWithBuffID(target.ID, PolarityHarmful, processor.rng)
	if !removed || buffID != 299 || !processor.statusManager.CanAct(target.ID) {
		t.Fatalf("stone removal = %v/%d", removed, buffID)
	}
}
