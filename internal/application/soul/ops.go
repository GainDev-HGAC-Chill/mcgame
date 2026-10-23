// Open-sourced by BaoLT

// Soul EXP progression and dissolve operations follow the Flash fraction and catalog rows.
package soul

import (
	"context"
	"fmt"
	"math"

	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

func (s *Service) loadChar(ctx context.Context, charID int64) (*domainchar.Character, error) {
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, err
	}
	if char == nil {
		return nil, ErrCharacterNotFound
	}
	return char, nil
}

func (s *Service) SoulLevelUp(ctx context.Context, charID int64, slot int) (map[string]any, int64, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	return s.soulLevelUp(ctx, charID, 0, slot)
}

func (s *Service) SoulLevelUpPet(ctx context.Context, charID, petID int64, slot int) (map[string]any, int64, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	if _, err := s.ownedPet(ctx, charID, petID); err != nil {
		return nil, 0, err
	}
	return s.soulLevelUp(ctx, charID, petID, slot)
}

func (s *Service) soulLevelUp(ctx context.Context, charID, petID int64, slot int) (map[string]any, int64, error) {
	var gem SoulSlot
	var found bool
	var charState *CharSoulState
	var petState *PetSoulState
	var err error
	if petID > 0 {
		petState, err = s.loadPetState(ctx, petID)
		if err == nil {
			gem, found = petState.Slots[slot]
		}
	} else {
		charState, err = s.loadCharState(ctx, charID)
		if err == nil {
			gem, found = charState.Slots[slot]
		}
	}
	if err != nil {
		return nil, 0, err
	}
	if !found {
		return nil, 0, ErrSlotEmpty
	}
	next, cost, err := s.levelUpStep(gem)
	if err != nil {
		return nil, 0, err
	}
	char, err := s.loadChar(ctx, charID)
	if err != nil {
		return nil, 0, err
	}
	if char.SoulExp < cost {
		return nil, 0, ErrInsufficientExp
	}
	previousSoulExp := char.SoulExp
	char.SoulExp -= cost
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.SoulExp = previousSoulExp
		return nil, 0, fmt.Errorf("soul: spend soul exp: %w", err)
	}
	gem.Sid = int(next.ID)
	gem.Exp = 0
	if petID > 0 {
		petState.Slots[slot] = gem
		err = s.savePetState(ctx, petID, petState)
	} else {
		charState.Slots[slot] = gem
		err = s.saveCharState(ctx, charID, charState)
	}
	if err != nil {
		char.SoulExp = previousSoulExp
		if compensationErr := s.charRepo.Update(ctx, char); compensationErr != nil {
			return nil, 0, fmt.Errorf("soul: save level-up state: %w; restore exp: %v", err, compensationErr)
		}
		return nil, 0, err
	}
	offset := 0
	if petID > 0 {
		offset = petSoulWireOffset
	}
	return slotWire(slot, gem, offset), char.SoulExp, nil
}

func (s *Service) levelUpStep(gem SoulSlot) (*models.PetSoulTemplate, int64, error) {
	current := s.petSoulTemplate(gem.Sid)
	if current == nil || current.UpExp <= 0 || current.UpExp != math.Trunc(current.UpExp) || current.Level != math.Trunc(current.Level) {
		return nil, 0, ErrSoulCatalogDrift
	}
	if int(current.Level) >= 10 {
		return nil, 0, ErrSoulMaxLevel
	}
	next := s.petSoulTemplate(gem.Sid + 1)
	if !sameSoulFamily(current, next) {
		return nil, 0, ErrSoulCatalogDrift
	}
	required := int64(current.UpExp)
	if gem.Exp < 0 || gem.Exp >= required {
		return nil, 0, ErrSoulCatalogDrift
	}
	return next, required - gem.Exp, nil
}

func sameSoulFamily(current, next *models.PetSoulTemplate) bool {
	return current != nil && next != nil && next.ID == current.ID+1 && int(next.Level) == int(current.Level)+1 && next.Level == math.Trunc(next.Level) && next.Name == current.Name && next.Type == current.Type && next.Color == current.Color && next.PropType == current.PropType
}

func (s *Service) PutInToExp(ctx context.Context, charID int64, slot int, fraction float64) (map[string]any, int64, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	if math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction <= 0 || fraction > 1 {
		return nil, 0, ErrInvalidFraction
	}
	state, err := s.loadCharState(ctx, charID)
	if err != nil {
		return nil, 0, err
	}
	gem, ok := state.Slots[slot]
	if !ok {
		return nil, 0, ErrSlotEmpty
	}
	tpl := s.petSoulTemplate(gem.Sid)
	if tpl == nil || tpl.UpExp <= 0 || tpl.UpExp != math.Trunc(tpl.UpExp) {
		return nil, 0, ErrSoulCatalogDrift
	}
	char, err := s.loadChar(ctx, charID)
	if err != nil {
		return nil, 0, err
	}
	remaining := int64(tpl.UpExp) - gem.Exp
	if remaining <= 0 || char.SoulExp <= 0 {
		return nil, 0, ErrInsufficientExp
	}
	available := min(remaining, char.SoulExp)
	amount := int64(math.Round(float64(available) * fraction))
	if amount < 1 {
		amount = 1
	}
	if amount > available {
		amount = available
	}
	previousSoulExp := char.SoulExp
	char.SoulExp -= amount
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.SoulExp = previousSoulExp
		return nil, 0, fmt.Errorf("soul: pour soul exp: %w", err)
	}
	gem.Exp += amount
	state.Slots[slot] = gem
	if err := s.saveCharState(ctx, charID, state); err != nil {
		char.SoulExp = previousSoulExp
		if compensationErr := s.charRepo.Update(ctx, char); compensationErr != nil {
			return nil, 0, fmt.Errorf("soul: save poured exp: %w; restore exp: %v", err, compensationErr)
		}
		return nil, 0, err
	}
	return slotWire(slot, gem, 0), char.SoulExp, nil
}

func (s *Service) TransformExp(ctx context.Context, charID int64, slot int) (map[string]any, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	state, err := s.loadCharState(ctx, charID)
	if err != nil {
		return nil, err
	}
	gem, ok := state.Slots[slot]
	if !ok {
		return nil, ErrSlotEmpty
	}
	if gem.Lock {
		return nil, ErrSlotLocked
	}
	char, err := s.loadChar(ctx, charID)
	if err != nil {
		return nil, err
	}
	expCredit, chipCredit, err := s.creditDissolve(char, []SoulSlot{gem})
	if err != nil {
		return nil, err
	}
	delete(state.Slots, slot)
	if err := s.persistDissolve(ctx, charID, char, state, expCredit, chipCredit); err != nil {
		return nil, err
	}
	return map[string]any{"f": 1, "index": slot, "soulExp": char.SoulExp, "chip": char.SoulPoints}, nil
}

func (s *Service) TransformAllExp(ctx context.Context, charID int64, threshold int) (map[string]any, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	if threshold < 1 || threshold > 3 {
		return nil, ErrInvalidThreshold
	}
	state, err := s.loadCharState(ctx, charID)
	if err != nil {
		return nil, err
	}
	char, err := s.loadChar(ctx, charID)
	if err != nil {
		return nil, err
	}
	selected := make([]SoulSlot, 0, len(state.Slots))
	indexes := make([]int, 0, len(state.Slots))
	for index, gem := range state.Slots {
		tpl := s.petSoulTemplate(gem.Sid)
		if gem.Lock {
			continue
		}
		if tpl == nil || tpl.Color != math.Trunc(tpl.Color) {
			return nil, ErrSoulCatalogDrift
		}
		if int(tpl.Color) >= threshold {
			continue
		}
		selected = append(selected, gem)
		indexes = append(indexes, index)
	}
	expCredit, chipCredit, err := s.creditDissolve(char, selected)
	if err != nil {
		return nil, err
	}
	for _, index := range indexes {
		delete(state.Slots, index)
	}
	if len(selected) > 0 {
		if err := s.persistDissolve(ctx, charID, char, state, expCredit, chipCredit); err != nil {
			return nil, err
		}
	}
	return map[string]any{"data": state.LoginObj(), "soulExp": char.SoulExp, "chip": char.SoulPoints}, nil
}

func (s *Service) TransformTempExp(ctx context.Context, charID int64, index int) (map[string]any, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	state, err := s.loadCharState(ctx, charID)
	if err != nil {
		return nil, err
	}
	sid, ok := state.TempBag[index]
	if !ok {
		return nil, ErrTempSlotEmpty
	}
	char, err := s.loadChar(ctx, charID)
	if err != nil {
		return nil, err
	}
	expCredit, chipCredit, err := s.creditDissolve(char, []SoulSlot{{Sid: sid}})
	if err != nil {
		return nil, err
	}
	delete(state.TempBag, index)
	if err := s.persistDissolve(ctx, charID, char, state, expCredit, chipCredit); err != nil {
		return nil, err
	}
	return map[string]any{"f": 1, "index": index, "soulExp": char.SoulExp, "chip": char.SoulPoints}, nil
}

func (s *Service) TransformAllTempExp(ctx context.Context, charID int64, threshold int) (map[string]any, error) {
	unlock := s.lockMutation(charID)
	defer unlock()
	if threshold < 1 || threshold > 3 {
		return nil, ErrInvalidThreshold
	}
	state, err := s.loadCharState(ctx, charID)
	if err != nil {
		return nil, err
	}
	char, err := s.loadChar(ctx, charID)
	if err != nil {
		return nil, err
	}
	selected := make([]SoulSlot, 0, len(state.TempBag))
	indexes := make([]int, 0, len(state.TempBag))
	for index, sid := range state.TempBag {
		tpl := s.petSoulTemplate(sid)
		if tpl == nil || tpl.Color != math.Trunc(tpl.Color) {
			return nil, ErrSoulCatalogDrift
		}
		if int(tpl.Color) >= threshold {
			continue
		}
		selected = append(selected, SoulSlot{Sid: sid})
		indexes = append(indexes, index)
	}
	expCredit, chipCredit, err := s.creditDissolve(char, selected)
	if err != nil {
		return nil, err
	}
	for _, index := range indexes {
		delete(state.TempBag, index)
	}
	if len(selected) > 0 {
		if err := s.persistDissolve(ctx, charID, char, state, expCredit, chipCredit); err != nil {
			return nil, err
		}
	}
	return map[string]any{"data": state.TempBagWire(), "soulExp": char.SoulExp, "chip": char.SoulPoints}, nil
}

func (s *Service) creditDissolve(char *domainchar.Character, gems []SoulSlot) (int64, int64, error) {
	var expCredit int64
	var chipCredit int64
	for _, gem := range gems {
		tpl := s.petSoulTemplate(gem.Sid)
		if tpl == nil || tpl.Exp < 0 || tpl.Exp != math.Trunc(tpl.Exp) || tpl.Chip < 0 || tpl.Chip != math.Trunc(tpl.Chip) || gem.Exp < 0 {
			return 0, 0, ErrSoulCatalogDrift
		}
		expCredit += int64(tpl.Exp) + gem.Exp
		chipCredit += int64(tpl.Chip)
	}
	char.SoulExp += expCredit
	char.SoulPoints += chipCredit
	return expCredit, chipCredit, nil
}

func (s *Service) persistDissolve(ctx context.Context, charID int64, char *domainchar.Character, state *CharSoulState, expCredit, chipCredit int64) error {
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.SoulExp -= expCredit
		char.SoulPoints -= chipCredit
		return fmt.Errorf("soul: credit dissolve: %w", err)
	}
	if err := s.saveCharState(ctx, charID, state); err != nil {
		char.SoulExp -= expCredit
		char.SoulPoints -= chipCredit
		if compensationErr := s.charRepo.Update(ctx, char); compensationErr != nil {
			return fmt.Errorf("soul: save dissolve state: %w; restore currency: %v", err, compensationErr)
		}
		return fmt.Errorf("soul: save dissolve state: %w", err)
	}
	return nil
}
