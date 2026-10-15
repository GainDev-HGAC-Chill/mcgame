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
