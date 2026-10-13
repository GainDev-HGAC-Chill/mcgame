// Cached equipment eligibility and displaced-item moves commit together against current player state.
package item

import (
	"context"
	"fmt"
	"math"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

type EquippedSlotUpdater interface {
	UpdateEquippedSlotsConditional(context.Context, int64, func(*character.Character, []*item.Item) ([]*item.Item, error)) (*character.Character, error)
}

type ResourceCharacterUpdater interface {
	UpdateCachedCharacterConditional(int64, func(*character.Character, []*item.Item) (bool, error)) (*character.Character, bool, error)
}

func (s *Service) equipItemAtomic(ctx context.Context, updater EquippedSlotUpdater, charID, itemID int64, slot item.EquipSlot) (*item.Item, error) {
	preliminary, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if preliminary == nil {
		return nil, pkgerrors.ErrItemNotFound
	}
	if preliminary.CharacterID != charID {
		return nil, pkgerrors.ErrItemNotOwned
	}
	var displaced *item.Item
	_, err = updater.UpdateEquippedSlotsConditional(ctx, charID, func(current *character.Character, inventory []*item.Item) ([]*item.Item, error) {
		var incoming *item.Item
		occupied := make(map[int]*item.Item)
		bag := make(map[int]bool)
		for _, it := range inventory {
			if it == nil {
				continue
			}
			if it.ID == itemID {
				incoming = it
			}
			if it.SlotType == item.SlotTypeEquipped {
				occupied[it.SlotIndex] = it
			}
			if it.SlotType == item.SlotTypeBag {
				bag[it.SlotIndex] = true
			}
		}
		if incoming == nil {
			return nil, pkgerrors.ErrItemNotFound
		}
		if incoming.CharacterID != charID {
			return nil, pkgerrors.ErrItemNotOwned
		}
		if !incoming.IsEquipment() || s.gameDataRec == nil {
			return nil, pkgerrors.ErrCannotEquip
		}
		if incoming.IsBroken() {
			return nil, pkgerrors.ErrItemBroken
		}
		tpl := s.gameDataRec.GetEquipment(incoming.TemplateID)
		if tpl == nil || math.IsNaN(tpl.ReqLevel) || math.IsInf(tpl.ReqLevel, 0) {
			return nil, pkgerrors.ErrCannotEquip
		}
		if float64(current.Level) < tpl.ReqLevel {
			return nil, pkgerrors.ErrInsufficientLevel
		}
		position := int(tpl.Position) - 1
		if position < 0 || position > 21 {
			return nil, pkgerrors.ErrCannotEquip
		}
		target := int(slot)
		if target < 0 {
			target = position
			if target == int(item.EquipSlotRing1) && occupied[7] != nil && occupied[8] == nil {
				target = 8
			}
		}
		if target != position && !(position == 7 && target == 8) {
			return nil, pkgerrors.ErrCannotEquip
		}
		if incoming.SlotType == item.SlotTypeEquipped && incoming.SlotIndex == target {
			return nil, nil
		}
		if incoming.SlotType != item.SlotTypeBag {
			return nil, pkgerrors.ErrCannotEquip
		}
		changes := make([]*item.Item, 0, 2)
		if existing := occupied[target]; existing != nil {
			empty := -1
			for i := 0; i < current.MaxBagSlots(); i++ {
				if !bag[i] {
					empty = i
					break
				}
			}
			if empty < 0 {
				return nil, pkgerrors.ErrInventoryFull
			}
			existing.SlotType, existing.SlotIndex = item.SlotTypeBag, empty
			displaced = existing
			changes = append(changes, existing)
		}
		incoming.SlotType, incoming.SlotIndex = item.SlotTypeEquipped, target
		return append(changes, incoming), nil
	})
	if err != nil {
		return nil, err
	}
	return displaced, nil
}

func (s *Service) SyncEquipmentResources(ctx context.Context, charID int64) (*character.Character, bool, error) {
	updater, ok := s.itemRepo.(ResourceCharacterUpdater)
	if !ok {
		return nil, false, fmt.Errorf("atomic resource patch unavailable")
	}
	bonuses := s.AggregateEquipmentStats(ctx, charID)
	return updater.UpdateCachedCharacterConditional(charID, func(current *character.Character, _ []*item.Item) (bool, error) {
		computed := *current
		computed.RecalculateStats()
		computed.ApplyEquipmentBonuses(bonuses)
		return current.SyncToEffectiveMax(computed.MaxHP, computed.MaxMP), nil
	})
}
