// Open-sourced by BaoLT

// Decoration ownership, equip, visibility, visual tier, expiry, and catalog-backed stat projection.
package dress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

var (
	ErrDecoShowInvalid          = errors.New("deco: invalid decoration")
	ErrDecoShowNotOwned         = errors.New("deco: decoration not owned")
	ErrDecoShowNotEquipped      = errors.New("deco: decoration not equipped")
	ErrDecoShowAlreadyOwned     = errors.New("deco: decoration already owned")
	ErrDecoShowInsufficientGold = errors.New("deco: not enough gold")
	ErrDecoShowInvalidLevel     = errors.New("deco: invalid show level")
)

type decoActiveFlag struct {
	Runes map[string]interface{} `json:"r"`
	Shows map[string]int64       `json:"s"`
}

type DecoShowMutationResult struct {
	State     *DecoHoleState
	SpentGold int64
	GoldAfter int64
}

func decodeDecoActiveFlag(raw string) decoActiveFlag {
	flag := decoActiveFlag{Runes: map[string]interface{}{}, Shows: map[string]int64{}}
	if raw == "" {
		return flag
	}
	var value map[string]interface{}
	if json.Unmarshal([]byte(raw), &value) != nil {
		return flag
	}
	if runes, ok := value["r"].(map[string]interface{}); ok {
		flag.Runes = runes
	}
	if shows, ok := value["s"].(map[string]interface{}); ok {
		for key, rawDue := range shows {
			if due, ok := intFromAny(rawDue); ok {
				flag.Shows[key] = int64(due)
			}
		}
	}
	return flag
}

func encodeDecoActiveFlag(flag decoActiveFlag) string {
	if flag.Runes == nil {
		flag.Runes = map[string]interface{}{}
	}
	if flag.Shows == nil {
		flag.Shows = map[string]int64{}
	}
	encoded, err := json.Marshal(flag)
	if err != nil {
		return "{\"r\":{},\"s\":{}}"
	}
	return string(encoded)
}

func decoShowOwned(flag decoActiveFlag, showID int, nowMs int64) bool {
	due, ok := flag.Shows[strconv.Itoa(showID)]
	return ok && (due == 1 || due > nowMs)
}

func (s *DecoHoleService) ActivateDecoShow(ctx context.Context, charID int64, showID, position int) (*DecoShowMutationResult, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()
	tpl, err := s.validDecoShow(showID, position)
	if err != nil {
		return nil, err
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("deco_show: load character: %w", err)
	}
	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	hole := s.ensureHolePosition(state, position)
	flag := decodeDecoActiveFlag(hole.ActiveFlag)
	nowMs := time.Now().UnixMilli()
	if decoShowOwned(flag, showID, nowMs) {
		return nil, ErrDecoShowAlreadyOwned
	}
	cost := int64(tpl.ActiveGold)
	if char.Gold < cost {
		return nil, ErrDecoShowInsufficientGold
	}
	originalGold := char.Gold
	char.Gold -= cost
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.Gold = originalGold
		return nil, fmt.Errorf("deco_show: persist gold deduction: %w", err)
	}
	due := int64(1)
	if int(tpl.T) > 1 {
		due = time.Now().Add(time.Duration(int64(tpl.T)) * time.Minute).UnixMilli()
	}
	flag.Shows[strconv.Itoa(showID)] = due
	hole.ActiveFlag = encodeDecoActiveFlag(flag)
	if hole.Did == 0 {
		hole.ID = int64(showID)
		hole.Did = int64(showID)
		hole.IsShow = 1
		if hole.ShowLvl < 1 {
			hole.ShowLvl = 1
		}
	}
	if err := s.saveState(ctx, charID, state); err != nil {
		char.Gold = originalGold
		if restoreErr := s.charRepo.Update(ctx, char); restoreErr != nil {
			return nil, fmt.Errorf("deco_show: persist ownership: %w; restore gold: %v", err, restoreErr)
		}
		return nil, fmt.Errorf("deco_show: persist ownership: %w", err)
	}
	return &DecoShowMutationResult{State: state, SpentGold: cost, GoldAfter: char.Gold}, nil
}

func (s *DecoHoleService) EquipDecoShow(ctx context.Context, charID int64, showID, position int) (*DecoHoleState, error) {
	return s.mutateDecoShowState(ctx, charID, func(state *DecoHoleState) error {
		if _, err := s.validDecoShow(showID, position); err != nil {
			return err
		}
		hole := s.ensureHolePosition(state, position)
		if !decoShowOwned(decodeDecoActiveFlag(hole.ActiveFlag), showID, time.Now().UnixMilli()) {
			return ErrDecoShowNotOwned
		}
		hole.ID = int64(showID)
		hole.Did = int64(showID)
		hole.IsShow = 1
		if hole.ShowLvl < 1 {
			hole.ShowLvl = 1
		}
		return nil
	})
}

func (s *DecoHoleService) EquipOffDecoShow(ctx context.Context, charID int64, showID, position int) (*DecoHoleState, error) {
	return s.mutateDecoShowState(ctx, charID, func(state *DecoHoleState) error {
		if _, err := s.validDecoShow(showID, position); err != nil {
			return err
		}
		hole := s.ensureHolePosition(state, position)
		if int(hole.Did) != showID {
			return ErrDecoShowNotEquipped
		}
		hole.ID = 0
		hole.Did = 0
		hole.IsShow = 0
		return nil
	})
}

func (s *DecoHoleService) SetDecoShowVisible(ctx context.Context, charID int64, position int, visible bool) (*DecoHoleState, error) {
	return s.mutateDecoShowState(ctx, charID, func(state *DecoHoleState) error {
		if position < decoHolePositionMin || position > decoHolePositionMax {
			return ErrDecoHoleInvalidPosition
		}
		hole := s.ensureHolePosition(state, position)
		showID := int(hole.Did)
		if showID <= 0 || !decoShowOwned(decodeDecoActiveFlag(hole.ActiveFlag), showID, time.Now().UnixMilli()) {
			return ErrDecoShowNotEquipped
		}
		if visible {
			hole.IsShow = 1
		} else {
			hole.IsShow = 0
		}
		return nil
	})
}

func (s *DecoHoleService) ChangeDecoShowLevel(ctx context.Context, charID int64, showLevel, position int) (*DecoHoleState, error) {
	return s.mutateDecoShowState(ctx, charID, func(state *DecoHoleState) error {
		if position < decoHolePositionMin || position > decoHolePositionMax {
			return ErrDecoHoleInvalidPosition
		}
		hole := s.ensureHolePosition(state, position)
		if hole.Did <= 0 || !decoShowOwned(decodeDecoActiveFlag(hole.ActiveFlag), int(hole.Did), time.Now().UnixMilli()) {
			return ErrDecoShowNotEquipped
		}
		holeTpl := s.gameDataMgr.GetDecoHole(int(hole.Hid))
		if holeTpl == nil {
			return ErrDecoShowInvalidLevel
		}
		maxLevel := 1
		if int(holeTpl.Level) >= 30 {
			maxLevel = 2
		}
		if int(holeTpl.Level) >= 50 {
			maxLevel = 3
		}
		if showLevel < 1 || showLevel > maxLevel {
			return ErrDecoShowInvalidLevel
		}
		hole.ShowLvl = showLevel
		return nil
	})
}

func (s *DecoHoleService) RefreshExpiredDecoShows(ctx context.Context, charID int64) (*DecoHoleState, map[string]interface{}, error) {
	var due map[string]interface{}
	state, err := s.mutateDecoShowState(ctx, charID, func(state *DecoHoleState) error {
		due = make(map[string]interface{})
		nowMs := time.Now().UnixMilli()
		for _, hole := range state.Positions {
			flag := decodeDecoActiveFlag(hole.ActiveFlag)
			for rawID, expiresAt := range flag.Shows {
				if expiresAt > 1 && expiresAt <= nowMs {
					delete(flag.Shows, rawID)
					if strconv.FormatInt(hole.Did, 10) == rawID {
						hole.ID = 0
						hole.Did = 0
						hole.IsShow = 0
					}
					continue
				}
				if expiresAt > 1 {
					due[rawID] = expiresAt
				}
			}
			hole.ActiveFlag = encodeDecoActiveFlag(flag)
		}
		return nil
	})
	return state, due, err
}

func (s *DecoHoleService) mutateDecoShowState(ctx context.Context, charID int64, mutation func(*DecoHoleState) error) (*DecoHoleState, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()
	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	if err := mutation(state); err != nil {
		return nil, err
	}
	if err := s.saveState(ctx, charID, state); err != nil {
		return nil, fmt.Errorf("deco_show: persist state: %w", err)
	}
	return state, nil
}

func (s *DecoHoleService) validDecoShow(showID, position int) (*models.DecoShowTemplate, error) {
	if position < decoHolePositionMin || position > decoHolePositionMax {
		return nil, ErrDecoHoleInvalidPosition
	}
	tpl := s.gameDataMgr.GetDecoShow(showID)
	if tpl == nil || int(tpl.Position) != position {
		return nil, ErrDecoShowInvalid
	}
	return tpl, nil
}

func (s *DecoHoleService) AggregateCharacterStatBonuses(ctx context.Context, charID int64) domainchar.EquipmentStatBonuses {
	bonuses := domainchar.NewEquipmentStatBonuses()
	if s == nil || s.gameDataMgr == nil {
		return bonuses
	}
	state, err := s.loadState(ctx, charID)
	if err != nil {
		return bonuses
	}
	nowMs := time.Now().UnixMilli()
	for _, hole := range state.Positions {
		if hole == nil {
			continue
		}
		if holeTpl := s.gameDataMgr.GetDecoHole(int(hole.Hid)); holeTpl != nil {
			applyDecoProps(&bonuses, [4]float64{holeTpl.PropType1, holeTpl.PropType2, holeTpl.PropType3, holeTpl.PropType4}, [4]float64{holeTpl.PropNum1, holeTpl.PropNum2, holeTpl.PropNum3, holeTpl.PropNum4}, false)
		}
		if hole.Did > 0 && decoShowOwned(decodeDecoActiveFlag(hole.ActiveFlag), int(hole.Did), nowMs) {
			if showTpl := s.gameDataMgr.GetDecoShow(int(hole.Did)); showTpl != nil && int(showTpl.Position) == hole.Position {
				applyDecoProps(&bonuses, [4]float64{showTpl.PropType1, showTpl.PropType2, showTpl.PropType3, showTpl.PropType4}, [4]float64{showTpl.PropNum1, showTpl.PropNum2, showTpl.PropNum3, showTpl.PropNum4}, int(showTpl.Per) == 1)
			}
		}
		for _, runeID := range hole.R {
			runeTpl := s.gameDataMgr.GetDecoRune(runeID)
			if runeTpl == nil || int(runeTpl.Kind) != 1 {
				continue
			}
			applyDecoProp(&bonuses, int(runeTpl.PropType), runeTpl.PropNum, int(runeTpl.Per) == 1)
		}
	}
	return bonuses
}

func applyDecoProps(bonuses *domainchar.EquipmentStatBonuses, types, values [4]float64, percent bool) {
	for index := range types {
		applyDecoProp(bonuses, int(types[index]), values[index], percent)
	}
}

func applyDecoProp(bonuses *domainchar.EquipmentStatBonuses, propID int, rawValue float64, percent bool) {
	if bonuses == nil || propID <= 0 || rawValue == 0 {
		return
	}
	statProp := domainchar.NormalizeCharacterPropType(propID)
	if statProp == 0 {
		return
	}
	value := domainchar.NormalizeRatePropValue(statProp, rawValue)
	if percent && domainchar.IsPercentScalableProp(statProp) {
		bonuses.AddPercent(statProp, int(math.Round(value)))
		return
	}
	if domainchar.UsesFloatBonus(statProp) {
		bonuses.AddFloat(statProp, value)
		return
	}
	bonuses.AddFlat(statProp, int(math.Round(value)))
}
