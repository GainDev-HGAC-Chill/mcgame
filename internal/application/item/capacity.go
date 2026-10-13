package item

import (
	"context"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	bankExpansionBook = 324
	bagExpansionBook  = 325
	petExpansionBook  = 326
)

func isCapacityBook(templateID int) bool {
	return templateID == bankExpansionBook || templateID == bagExpansionBook || templateID == petExpansionBook
}

func (s *Service) useCapacityBooks(ctx context.Context, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate, count, targetType int) (map[string]interface{}, error) {
	if tpl == nil || int(tpl.Type) != 500 || int(tpl.UseType) != 1 || (targetType != 0 && targetType != 1) {
		return nil, pkgerrors.ErrCannotUseItem
	}
	if char.Level < int(tpl.ReqLevel) {
		return nil, pkgerrors.ErrInsufficientLevel
	}
	if count <= 0 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	if it.StackCount < count {
		return nil, pkgerrors.ErrInsufficientItems
	}

	updates := map[string]interface{}{}
	current, limit := 0, 0
	switch it.TemplateID {
	case bankExpansionBook:
		current, limit = char.BankSlotNum, character.MaxBankTabs
		if current <= 0 && char.BankSlots > 0 {
			current = (char.BankSlots + character.SlotsPerBank - 1) / character.SlotsPerBank
		}
	case bagExpansionBook:
		current, limit = char.BagSlotNum, character.MaxBagTabs
		if current <= 0 && char.BagSlots > 0 {
			current = (char.BagSlots + character.SlotsPerBag - 1) / character.SlotsPerBag
		}
	case petExpansionBook:
		current, limit = char.PetMaxNum, domainpet.MaxPets
		if current <= 0 {
			current = char.PetSlots
		}
	default:
		return nil, pkgerrors.ErrCannotUseItem
	}
	if current < 0 || current >= limit || count > limit-current {
		return nil, pkgerrors.ErrInventoryFull
	}

	remaining := it.StackCount - count
	if remaining == 0 {
		if err := s.itemRepo.Delete(ctx, it.ID); err != nil {
			return nil, err
		}
	} else if err := s.itemRepo.UpdateStack(ctx, it.ID, remaining); err != nil {
		return nil, err
	}
	it.StackCount = remaining

	switch it.TemplateID {
	case bankExpansionBook:
		char.SetBankSlotNum(current + count)
		updates["bankSlotNum"] = char.BankSlotNum
	case bagExpansionBook:
		char.SetBagSlotNum(current + count)
		updates["bagSlotNum"] = char.BagSlotNum
	case petExpansionBook:
		char.PetMaxNum = current + count
		char.PetSlots = char.PetMaxNum
		updates["petMaxNum"] = char.PetMaxNum
	}

	return map[string]interface{}{
		"capacity_updates": updates,
		"consumed":         count,
		"remaining":        remaining,
		"item_id":          it.ID,
		"used_item":        it,
	}, nil
}
