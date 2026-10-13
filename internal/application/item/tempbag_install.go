package item

import (
	"context"
	"errors"
	"time"

	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

const tempBagExpiresAtKey = "tempBagExpiresAt"

type TempBagInstallResult struct {
	Installed *domainitem.Item
	Replaced  *domainitem.Item
}

func tempBagInstalledIndex(slot int) int {
	return -1000 - slot
}

func tempBagInstalledSlot(index int) (int, bool) {
	slot := -1000 - index
	return slot, slot >= 1 && slot <= 5
}

func (s *Service) InstallTempBag(ctx context.Context, charID int64, fromSID, toSID int, unlockedSlots int, now time.Time) (*TempBagInstallResult, error) {
	if s.itemRepo == nil || s.gameDataRec == nil || fromSID < 2101 || fromSID > 2310 || toSID < 80 || toSID > 84 {
		return nil, pkgerrors.ErrInvalidInput
	}
	slot := toSID - 79
	if slot > unlockedSlots {
		return nil, pkgerrors.ErrInvalidInput
	}
	newBag, err := s.itemRepo.FindBySlot(ctx, charID, domainitem.SlotTypeBag, fromSID-2101)
	if err != nil {
		return nil, err
	}
	tpl := s.gameDataRec.GetItem(newBag.TemplateID)
	if tpl == nil || int(tpl.Type) != 517 || int(tpl.ProplNum) <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	installedIndex := tempBagInstalledIndex(slot)
	replaced, err := s.itemRepo.FindBySlot(ctx, charID, domainitem.SlotTypeTempBag, installedIndex)
	if err != nil && !errors.Is(err, pkgerrors.ErrItemNotFound) {
		return nil, err
	}
	items, err := s.itemRepo.FindByCharacterID(ctx, charID)
	if err != nil {
		return nil, err
	}
	capacity := s.tempBagCapacity(items, now)
	if replaced != nil {
		capacity -= s.activeTempBagCapacity(replaced, now)
	}
	capacity += int(tpl.ProplNum)
	for _, it := range items {
		if it != nil && it.SlotType == domainitem.SlotTypeTempBag && it.SlotIndex >= 0 && it.SlotIndex < 100 && it.SlotIndex > capacity {
			return nil, pkgerrors.ErrInvalidInput
		}
	}

	sourceIndex := newBag.SlotIndex
	stagingIndex := tempBagInstalledIndex(99)
	if replaced != nil {
		if err := s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeTempBag, stagingIndex); err != nil {
			return nil, err
		}
		if err := s.itemRepo.MoveItem(ctx, replaced.ID, domainitem.SlotTypeBag, sourceIndex); err != nil {
			_ = s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeBag, sourceIndex)
			return nil, err
		}
	} else if err := s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeTempBag, installedIndex); err != nil {
		return nil, err
	}
	if replaced != nil {
		if err := s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeTempBag, installedIndex); err != nil {
			_ = s.itemRepo.MoveItem(ctx, replaced.ID, domainitem.SlotTypeTempBag, installedIndex)
			_ = s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeBag, sourceIndex)
			return nil, err
		}
		replaced.SlotType = domainitem.SlotTypeBag
		replaced.SlotIndex = sourceIndex
	}
	newBag.SlotType = domainitem.SlotTypeTempBag
	newBag.SlotIndex = installedIndex
	if tpl.T > 0 {
		if newBag.Properties == nil {
			newBag.Properties = make(map[string]interface{})
		}
		if _, active := tempBagExpiry(newBag); !active {
			newBag.Properties[tempBagExpiresAtKey] = now.Add(time.Duration(tpl.T) * time.Minute).UTC().Format(time.RFC3339Nano)
		}
	}
	if err := s.itemRepo.Update(ctx, newBag); err != nil {
		if replaced != nil {
			_ = s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeTempBag, stagingIndex)
			_ = s.itemRepo.MoveItem(ctx, replaced.ID, domainitem.SlotTypeTempBag, installedIndex)
		}
		_ = s.itemRepo.MoveItem(ctx, newBag.ID, domainitem.SlotTypeBag, sourceIndex)
		return nil, err
	}
	return &TempBagInstallResult{Installed: newBag, Replaced: replaced}, nil
}

func (s *Service) tempBagCapacity(items []*domainitem.Item, now time.Time) int {
	capacity := 0
	for _, it := range items {
		capacity += s.activeTempBagCapacity(it, now)
	}
	return capacity
}

func (s *Service) activeTempBagCapacity(it *domainitem.Item, now time.Time) int {
	if it == nil || it.SlotType != domainitem.SlotTypeTempBag {
		return 0
	}
	if _, installed := tempBagInstalledSlot(it.SlotIndex); !installed {
		return 0
	}
	if expiresAt, ok := tempBagExpiry(it); ok && !now.Before(expiresAt) {
		return 0
	}
	if s.gameDataRec == nil {
		return 0
	}
	tpl := s.gameDataRec.GetItem(it.TemplateID)
	if tpl == nil || int(tpl.Type) != 517 {
		return 0
	}
	return int(tpl.ProplNum)
}

func tempBagExpiry(it *domainitem.Item) (time.Time, bool) {
	if it == nil || it.Properties == nil {
		return time.Time{}, false
	}
	raw, ok := it.Properties[tempBagExpiresAtKey].(string)
	if !ok || raw == "" {
		return time.Time{}, false
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, raw)
	return expiresAt, err == nil
}
