// Open-sourced by BaoLT

// Monster Heart stat projection accepts only catalog-backed, ownership-bounded equipped hearts.
package statfeature

import (
	"math"

	appmonsterheart "mcgame-server/internal/application/monsterheart"
	domainchar "mcgame-server/internal/domain/character"
)

var monsterHeartPercentProps = map[int]struct{}{
	59: {},
	60: {},
	62: {},
	63: {},
}

func (s *Service) applyMonsterHeartBonus(bonuses *domainchar.EquipmentStatBonuses, rawState map[string]interface{}) {
	if bonuses == nil || len(rawState) == 0 || s.gameDataManager == nil {
		return
	}
	state := appmonsterheart.DecodeState(rawState)
	owned, valid := s.monsterHeartOwnedCounts(state)
	if !valid {
		return
	}
	for box := 1; box <= appmonsterheart.BoxCount; box++ {
		for hole := 1; hole <= appmonsterheart.HoleCount; hole++ {
			heartID := state.GetHoleItem(box, hole)
			if heartID <= 0 || owned[heartID] <= 0 {
				continue
			}
			heart := s.gameDataManager.GetCreatureHeart(heartID)
			point := s.gameDataManager.GetCreatureHeartPoint(box*10 + hole)
			if heart == nil || point == nil || int(point.Type) > 0 && int(point.Type) != int(heart.Type) {
				continue
			}
			owned[heartID]--
			applyMonsterHeartPropBonus(bonuses, int(heart.PropType), heart.Propnum)
		}
	}
}

func (s *Service) monsterHeartOwnedCounts(state *appmonsterheart.State) (map[int]int, bool) {
	owned := make(map[int]int)
	seenEntries := make(map[int]struct{})
	for category, entries := range state.Bag {
		for entryID, entry := range entries {
			if entryID <= 0 || entry.ID != entryID || entry.N <= 0 {
				return nil, false
			}
			if _, exists := seenEntries[entryID]; exists {
				return nil, false
			}
			seenEntries[entryID] = struct{}{}
			heart := s.gameDataManager.GetCreatureHeart(entry.ItemID)
			if heart == nil || category != monsterHeartCategory(int(heart.Type)) {
				continue
			}
			owned[entry.ItemID] += entry.N
		}
	}
	return owned, true
}

func monsterHeartCategory(heartType int) string {
	categories := [...]string{"", "ren", "shou", "zhi", "ji", "mo", "lon", "te"}
	if heartType < 1 || heartType >= len(categories) {
		return ""
	}
	return categories[heartType]
}

func applyMonsterHeartPropBonus(bonuses *domainchar.EquipmentStatBonuses, propID int, rawValue float64) {
	if bonuses == nil || rawValue <= 0 {
		return
	}
	divisor := 10000.0
	if _, ok := monsterHeartPercentProps[propID]; ok {
		divisor = 100.0
	}
	value := rawValue / divisor
	statProp := domainchar.NormalizeCharacterPropType(propID)
	if statProp == 0 {
		return
	}
	if domainchar.UsesFloatBonus(statProp) {
		bonuses.AddFloat(statProp, value)
		return
	}
	bonuses.AddFlat(statProp, int(math.Round(value)))
}
