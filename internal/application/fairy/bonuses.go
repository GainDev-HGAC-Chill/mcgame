// Open-sourced by BaoLT

package fairy

import (
	"context"
	"math"

	statfeatureapp "mcgame-server/internal/application/statfeature"
	domainchar "mcgame-server/internal/domain/character"

	"go.uber.org/zap"
)

const passiveFairyFamilyCount = 6

var passiveFairyBuffs = [passiveFairyFamilyCount][skillMaxLevel]int{
	{2417, 2418, 2419},
	{2420, 2421, 2422},
	{2423, 2424, 2425},
	{2426, 2427, 2428},
	{2429, 2430, 2431},
	{2432, 2433, 2434},
}

func (s *Service) AggregateCharacterStatBonuses(ctx context.Context, charID int64) domainchar.EquipmentStatBonuses {
	bonuses := domainchar.NewEquipmentStatBonuses()
	if s == nil || s.featureRepo == nil {
		return bonuses
	}

	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		s.logger.Warn("fairy: aggregate bonuses failed", zap.Int64("character_id", charID), zap.Error(err))
		return bonuses
	}

	guarding := guardingFairy(state)
	if guarding == nil {
		return bonuses
	}

	applyFairyAttributeBonuses(&bonuses, guarding)
	s.applyFairyPassiveBonuses(&bonuses, guarding)
	return bonuses
}

func guardingFairy(state *State) *Fairy {
	if state == nil {
		return nil
	}
	var guarding *Fairy
	for _, candidate := range state.Fairies {
		if candidate == nil || candidate.State != fairyStateGuard {
			continue
		}
		if guarding != nil {
			return nil
		}
		guarding = candidate
	}
	return guarding
}

func applyFairyAttributeBonuses(bonuses *domainchar.EquipmentStatBonuses, f *Fairy) {
	if bonuses == nil || f == nil {
		return
	}
	level := expToLv(f.Exp)
	growthLevel := gexpToLv(f.Gexp)
	qualityIndex := (growthLevel - 1) / 10
	if qualityIndex >= len(FairyColorEffect) {
		qualityIndex = len(FairyColorEffect) - 1
	}
	qualityPct := float64(FairyColorEffect[qualityIndex])
	satiety := math.Round(float64(clampInt(f.Doh, 0, satietyMax)) / 100)
	satietyFactor := 1 - math.Max(75-satiety, 0)/75

	bonuses.AddFlat(domainchar.PropStrength, fairyOwnerAttribute(f.Sta, f.StaG, level, growthLevel, qualityPct, satietyFactor))
	bonuses.AddFlat(domainchar.PropStamina, fairyOwnerAttribute(f.Ste, f.SteG, level, growthLevel, qualityPct, satietyFactor))
	bonuses.AddFlat(domainchar.PropAgility, fairyOwnerAttribute(f.Agi, f.AgiG, level, growthLevel, qualityPct, satietyFactor))
	bonuses.AddFlat(domainchar.PropIntelligence, fairyOwnerAttribute(f.Inte, f.InteG, level, growthLevel, qualityPct, satietyFactor))
	bonuses.AddFlat(domainchar.PropSpirit, fairyOwnerAttribute(f.Ener, f.EnerG, level, growthLevel, qualityPct, satietyFactor))
}

func fairyOwnerAttribute(base, growth float64, level, growthLevel int, qualityPct, satietyFactor float64) int {
	raw := base + float64(level-1)*(growth+0.2*float64(growthLevel))
	if math.IsNaN(raw) || math.IsInf(raw, 0) || raw <= 0 || qualityPct <= 0 || satietyFactor <= 0 {
		return 0
	}
	return int(math.Round(raw * qualityPct * satietyFactor / 100))
}

func (s *Service) applyFairyPassiveBonuses(bonuses *domainchar.EquipmentStatBonuses, f *Fairy) {
	if bonuses == nil || f == nil || s.skillData == nil || s.buffData == nil {
		return
	}
	passive := domainchar.NewEquipmentStatBonuses()
	seen := make(map[int]struct{}, passiveFairyFamilyCount)
	for _, skillID := range f.SkillFlag {
		family, level, canonical := fairySkillFamily(skillID)
		if !canonical || family >= passiveFairyFamilyCount {
			continue
		}
		if _, duplicate := seen[family]; duplicate {
			return
		}
		seen[family] = struct{}{}

		skill := s.skillData.GetSkill(skillID)
		kind, validKind := skillTemplateKind(skill)
		catalogLevel, validLevel := skillTemplateLevel(skill)
		expectedBuffID := passiveFairyBuffs[family][level-1]
		if skill == nil {
			return
		}
		buffID, validBuffID := exactCatalogInt(skill.BuffID)
		if int(skill.ID) != skillID || !validKind || kind != 2 || !validLevel || catalogLevel != level || !validBuffID || buffID != expectedBuffID {
			return
		}
		buff := s.buffData.GetBuff(expectedBuffID)
		if buff == nil || int(buff.ID) != expectedBuffID || buff.PercentFlag != 1 {
			return
		}
		statfeatureapp.ApplyBuffTemplateBonus(&passive, buff)
	}
	mergeFairyBonuses(bonuses, passive)
}

func mergeFairyBonuses(dst *domainchar.EquipmentStatBonuses, src domainchar.EquipmentStatBonuses) {
	for prop, value := range src.Flat {
		dst.AddFlat(prop, value)
	}
	for prop, value := range src.Percent {
		dst.AddPercent(prop, value)
	}
	for prop, value := range src.Float {
		dst.AddFloat(prop, value)
	}
	for prop, value := range src.FinalAdds {
		dst.AddFinal(prop, value)
	}
}
