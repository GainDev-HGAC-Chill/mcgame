// Pet snapshots detach persisted dynamic state and private computed bonus maps.
// The caller must own the source or hold its owner's lock while cloning.
package pet

import "mcgame-server/pkg/valuecopy"

func (p *Pet) Clone() *Pet {
	if p == nil {
		return nil
	}
	result := *p
	result.Property = valuecopy.Copy(p.Property)
	result.CreatureData = valuecopy.Copy(p.CreatureData)
	result.SoulInfo = valuecopy.Copy(p.SoulInfo)
	result.skillBonuses.Scaled = valuecopy.Copy(p.skillBonuses.Scaled)
	result.skillBonuses.Direct = valuecopy.Copy(p.skillBonuses.Direct)
	result.equipmentBonuses.Flat = valuecopy.Copy(p.equipmentBonuses.Flat)
	result.equipmentBonuses.Percent = valuecopy.Copy(p.equipmentBonuses.Percent)
	result.equipmentBonuses.Float = valuecopy.Copy(p.equipmentBonuses.Float)
	result.equipmentBonuses.Feature = valuecopy.Copy(p.equipmentBonuses.Feature)
	return &result
}
