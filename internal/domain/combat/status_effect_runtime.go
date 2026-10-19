// Runtime status helpers apply effective stats, mitigation, shields, dispels, and expiry.
package combat

import (
	"math/rand"
	"sort"

	"mcgame-server/internal/domain/stats"
)

func (m *StatusEffectManager) EffectiveParticipant(p *Participant) *Participant {
	if p == nil {
		return nil
	}
	effective := *p
	cloned := stats.NewEffectiveStats()
	for id, value := range effective.EffStats.All() {
		cloned.Set(id, value)
	}
	effective.EffStats = cloned
	for _, effect := range m.effects[p.ID] {
		for _, modifier := range effect.Modifiers {
			applyStatModifier(&effective, modifier)
		}
	}
	if effective.EffStats.Get(stats.PropHP) > 0 {
		effective.EffStats.Set(stats.PropPhysAtk, int64(effective.Attack))
		effective.EffStats.Set(stats.PropMagAtk, int64(effective.MagicAttack))
		effective.EffStats.Set(stats.PropPhysDef, int64(effective.Defense))
		effective.EffStats.Set(stats.PropMagDef, int64(effective.MagicDefense))
		effective.EffStats.Set(stats.PropSpeed, int64(effective.Speed))
		effective.EffStats.Set(stats.PropDodge, int64(effective.Dodge))
		effective.EffStats.Set(stats.PropAccuracy, int64(effective.Hit))
		effective.EffStats.Set(stats.PropCritRate, int64(effective.Critical)*10)
		effective.EffStats.Set(stats.PropCombo, int64(effective.Combo)*10)
		effective.EffStats.Set(stats.PropPhysDmgCut, int64(effective.ReduceHurt1)*10)
		effective.EffStats.Set(stats.PropMagDmgCut, int64(effective.ReduceHurt2)*10)
	}
	return &effective
}

func applyStatModifier(p *Participant, modifier StatModifier) {
	apply := func(value int) int {
		return max(0, int(float64(value)*(1+modifier.Percent))+modifier.Flat)
	}
	switch modifier.Stat {
	case StatAttack:
		p.Attack = apply(p.Attack)
	case StatMagicAttack:
		p.MagicAttack = apply(p.MagicAttack)
	case StatDefense:
		p.Defense = apply(p.Defense)
	case StatMagicDefense:
		p.MagicDefense = apply(p.MagicDefense)
	case StatSpeed:
		p.Speed = apply(p.Speed)
	case StatDodge:
		p.Dodge = apply(p.Dodge)
	case StatAccuracy:
		p.Hit = apply(p.Hit)
	case StatCombo:
		p.Combo = apply(p.Combo)
	case StatCritical:
		p.Critical = apply(p.Critical)
	case StatReducePhysical:
		p.ReduceHurt1 = apply(p.ReduceHurt1)
	case StatReduceMagic:
		p.ReduceHurt2 = apply(p.ReduceHurt2)
	case StatDebuffResistance:
		base := p.EffStats.Get(stats.PropResistDebuff)
		p.EffStats.Set(stats.PropResistDebuff, max(0, int64(float64(base)*(1+modifier.Percent))+int64(modifier.Flat)))
	}
}

func (m *StatusEffectManager) RemoveFamily(participantID, family string) {
	effects := m.effects[participantID]
	remaining := make([]*StatusEffect, 0, len(effects))
	for _, effect := range effects {
		if effect.Family != family {
			remaining = append(remaining, effect)
		}
	}
	m.effects[participantID] = remaining
}

func (m *StatusEffectManager) IncomingDamageModifier(participantID string) float64 {
	modifier := 1.0
	for _, effect := range m.effects[participantID] {
		for _, stat := range effect.Modifiers {
			if stat.Stat == StatIncomingDamage {
				modifier *= 1 + stat.Percent
			}
		}
	}
	return max(0.0, modifier)
}

func (m *StatusEffectManager) ApplyDamage(participantID string, damage int, magic bool) int {
	if damage <= 0 {
		return 0
	}
	for _, effect := range m.effects[participantID] {
		if effect.Control.Invulnerable {
			return 0
		}
	}
	damage = int(float64(damage) * m.IncomingDamageModifier(participantID))
	for _, effect := range m.effects[participantID] {
		if effect.Shield == nil || effect.Shield.Hits <= 0 || effect.Shield.Pool <= 0 {
			continue
		}
		if effect.Shield.MagicOnly && !magic {
			continue
		}
		absorbed := min(damage, effect.Shield.Pool)
		effect.Shield.Pool -= absorbed
		effect.Shield.Hits--
		damage -= absorbed
		if damage == 0 {
			break
		}
	}
	m.OnDamageTaken(participantID, damage)
	return damage
}

func (m *StatusEffectManager) RemoveRandomEffect(participantID string, polarity EffectPolarity, rng *rand.Rand) bool {
	effects := m.effects[participantID]
	eligible := make([]int, 0)
	for i, effect := range effects {
		if effect.Removable && effect.Polarity == polarity {
			eligible = append(eligible, i)
		}
	}
	if len(eligible) == 0 {
		return false
	}
	index := eligible[rng.Intn(len(eligible))]
	m.effects[participantID] = append(effects[:index], effects[index+1:]...)
	return true
}

func (m *StatusEffectManager) RemoveAllEffects(participantID string, polarity EffectPolarity) int {
	effects := m.effects[participantID]
	remaining := make([]*StatusEffect, 0, len(effects))
	removed := 0
	for _, effect := range effects {
		if effect.Removable && effect.Polarity == polarity {
			removed++
			continue
		}
		remaining = append(remaining, effect)
	}
	m.effects[participantID] = remaining
	return removed
}

func (m *StatusEffectManager) TauntTarget(participantID string) string {
	for _, effect := range m.effects[participantID] {
		if effect.Control.TauntSourceID != "" {
			return effect.Control.TauntSourceID
		}
	}
	return ""
}

func (m *StatusEffectManager) ProcessTurnEnd(participantID string) {
	effects := m.effects[participantID]
	remaining := make([]*StatusEffect, 0, len(effects))
	for _, effect := range effects {
		if effect.Family != "" && effect.Tick() {
			continue
		}
		remaining = append(remaining, effect)
	}
	m.effects[participantID] = remaining
}

func (m *StatusEffectManager) SortedEffects(participantID string) []*StatusEffect {
	effects := append([]*StatusEffect(nil), m.effects[participantID]...)
	sort.SliceStable(effects, func(i, j int) bool {
		return effects[i].Family < effects[j].Family
	})
	return effects
}
