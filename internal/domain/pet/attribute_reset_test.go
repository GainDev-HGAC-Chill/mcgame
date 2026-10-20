package pet

import "testing"

func TestResetAllocatedAttributePointsRejectsMissingHistoryWithOtherEx(t *testing.T) {
	p := &Pet{AttrPoints: 3, DistributedAttrPoints: 5, AptStrengthEx: 5, AptEnergyEx: 1, Property: map[string]interface{}{}}
	if n, ok := p.ResetAllocatedAttributePoints(); ok || n != 0 {
		t.Fatalf("should refuse ambiguous history")
	}
	if p.AptStrengthEx != 5 || p.AptEnergyEx != 1 || p.DistributedAttrPoints != 5 {
		t.Fatal("changed data")
	}
}

func TestResetAllocatedAttributePointsPreservesNonAllocatedBonus(t *testing.T) {
	p := &Pet{AttrPoints: 10, AptEnergyEx: 3, Property: map[string]interface{}{}}
	if got := p.AllocateAttributePoints(4, 2, 0, 0, 0); got != 6 {
		t.Fatalf("spent %d", got)
	}
	got, ok := p.ResetAllocatedAttributePoints()
	if !ok || got != 6 {
		t.Fatalf("refund %d, ok=%v", got, ok)
	}
	if p.AttrPoints != 10 || p.DistributedAttrPoints != 0 || p.AptStrengthEx != 0 || p.AptAgilityEx != 0 || p.AptEnergyEx != 3 {
		t.Fatalf("lost bonus: %+v", p)
	}
}

func TestResetAllocatedAttributePointsLegacyWithoutProperty(t *testing.T) {
	p := &Pet{AttrPoints: 1, DistributedAttrPoints: 4, AptStrengthEx: 2, AptEnergyEx: 2}
	refunded, ok := p.ResetAllocatedAttributePoints()
	if !ok || refunded != 4 || p.AttrPoints != 5 || p.AptStrengthEx != 0 || p.AptEnergyEx != 0 || p.DistributedAttrPoints != 0 {
		t.Fatalf("legacy reset incorrect: %+v; refunded=%d ok=%v", p, refunded, ok)
	}
}

func TestResetAllocatedAttributePointsRejectsUntrackedPartialBeforeItemUse(t *testing.T) {
	p := &Pet{AttrPoints: 4, DistributedAttrPoints: 3, AptStrengthEx: 8, AptAgilityEx: 1, Property: map[string]interface{}{}}
	refunded, ok := p.ResetAllocatedAttributePoints()
	if ok || refunded != 0 || p.AttrPoints != 4 || p.AptStrengthEx != 8 {
		t.Fatalf("ambiguous partial allocation should not change pet: %+v", p)
	}
}
