package pet

import (
	"time"
)

// allocatedPetAttributePointsKey stores only points allocated by the player.
// Other effects also add to Apt*Ex, so never clear them without a ledger.
const allocatedPetAttributePointsKey = "_allocated_pet_attribute_points_v1"

var allocatedPetAttributeKeys = [...]string{"strength", "agility", "stamina", "intelligence", "energy"}

func (p *Pet) allocatedAttributePointLedger() ([5]int, bool) {
	var points [5]int
	if p.Property == nil {
		return points, false
	}
	raw, exists := p.Property[allocatedPetAttributePointsKey]
	if !exists {
		return points, false
	}
	fields, ok := raw.(map[string]interface{})
	if !ok {
		return points, true
	} // malformed ledger: caller will reject
	for i, key := range allocatedPetAttributeKeys {
		v, ok := petSceneIntValue(fields[key])
		if !ok || v < 0 || v > 1_000_000_000 {
			return [5]int{}, true // invalid ledger: do not guess
		}
		points[i] = int(v)
	}
	return points, true
}

func (p *Pet) recordAllocatedAttributePoints(delta [5]int) {
	if p.Property == nil {
		p.Property = map[string]interface{}{}
	}
	current, _ := p.allocatedAttributePointLedger()
	next := make(map[string]interface{}, 5)
	for i, key := range allocatedPetAttributeKeys {
		next[key] = current[i] + delta[i]
	}
	p.Property[allocatedPetAttributePointsKey] = next
}

// ResetAllocatedAttributePoints returns the number of refunded attribute points.
// For pets created before tracking, only an unambiguous all-allocated Apt*Ex
// snapshot can be reset; mixed legacy data must be reconciled manually.
func (p *Pet) ResetAllocatedAttributePoints() (int, bool) {
	if p == nil || p.DistributedAttrPoints <= 0 || p.AttrPoints < 0 {
		return 0, false
	}
	current := [5]int{p.AptStrengthEx, p.AptAgilityEx, p.AptStaminaEx, p.AptIntelligenceEx, p.AptEnergyEx}
	for _, value := range current {
		if value < 0 {
			return 0, false
		}
	}
	allocated, tracked := p.allocatedAttributePointLedger()
	if !tracked {
		total := 0
		for _, value := range current {
			total += value
		}
		if total != p.DistributedAttrPoints {
			return 0, false
		}
		allocated = current
	}
	total := 0
	for i, value := range allocated {
		if value < 0 || value > current[i] {
			return 0, false
		}
		total += value
	}
	if total != p.DistributedAttrPoints {
		return 0, false
	}
	p.AptStrengthEx -= allocated[0]
	p.AptAgilityEx -= allocated[1]
	p.AptStaminaEx -= allocated[2]
	p.AptIntelligenceEx -= allocated[3]
	p.AptEnergyEx -= allocated[4]
	p.AttrPoints += total
	p.DistributedAttrPoints = 0
	if p.Property == nil {
		p.Property = make(map[string]interface{})
	}
	p.Property[allocatedPetAttributePointsKey] = map[string]interface{}{
		"strength": 0, "agility": 0, "stamina": 0, "intelligence": 0, "energy": 0,
	}
	p.RecalculateStats()
	p.UpdatedAt = time.Now()
	return total, true
}
