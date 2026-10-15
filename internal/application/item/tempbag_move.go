package item

import (
	"context"

	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) MoveTempBagItemToBag(ctx context.Context, charID int64, tempIndex int) (*domainitem.Item, error) {
	if tempIndex <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	it, err := s.itemRepo.FindBySlot(ctx, charID, domainitem.SlotTypeTempBag, tempIndex)
	if err != nil {
		return nil, err
	}
	if it == nil || it.CharacterID != charID {
		return nil, pkgerrors.ErrItemNotOwned
	}
	maxBag := s.getMaxSlots(ctx, charID, domainitem.SlotTypeBag)
	emptySlot, err := s.itemRepo.FindFirstEmptySlot(ctx, charID, domainitem.SlotTypeBag, maxBag)
	if err != nil {
		return nil, err
	}
	if err := s.itemRepo.MoveItem(ctx, it.ID, domainitem.SlotTypeBag, emptySlot); err != nil {
		return nil, err
	}
	it.SlotType = domainitem.SlotTypeBag
	it.SlotIndex = emptySlot
	return it, nil
}

// MoveTempBagItemToSlot implements the Flash addItemFromTemp/addMXFromTemp
// contract. A zero count moves the complete stack; a positive count splits it.
func (s *Service) MoveTempBagItemToSlot(ctx context.Context, charID int64, tempIndex int, toSlotType domainitem.SlotType, toSlotIndex, count int) (*domainitem.Item, error) {
	if tempIndex <= 0 || toSlotIndex < 0 || (toSlotType != domainitem.SlotTypeBag && toSlotType != domainitem.SlotTypeQuestBag && toSlotType != domainitem.SlotTypePetItemBag) {
		return nil, pkgerrors.ErrInvalidInput
	}
	source, err := s.itemRepo.FindBySlot(ctx, charID, domainitem.SlotTypeTempBag, tempIndex)
	if err != nil {
		return nil, err
	}
	if source == nil || source.CharacterID != charID {
		return nil, pkgerrors.ErrItemNotOwned
	}
	if count < 0 || count > source.StackCount {
		return nil, pkgerrors.ErrCannotStack
	}
	if count > 0 && count < source.StackCount {
		_, destination, err := s.MoveItemPartial(ctx, charID, source.ID, toSlotType, toSlotIndex, count)
		return destination, err
	}

	destination, err := s.itemRepo.FindBySlot(ctx, charID, toSlotType, toSlotIndex)
	if err != nil && !pkgerrors.Is(err, pkgerrors.ErrItemNotFound) {
		return nil, err
	}
	if destination != nil && !source.CanStack(destination) {
		return nil, pkgerrors.ErrSlotOccupied
	}
	if err := s.MoveItem(ctx, charID, source.ID, toSlotType, toSlotIndex); err != nil {
		return nil, err
	}
	if destination != nil {
		return s.itemRepo.FindBySlot(ctx, charID, toSlotType, toSlotIndex)
	}
	source.SlotType = toSlotType
	source.SlotIndex = toSlotIndex
	return source, nil
}
