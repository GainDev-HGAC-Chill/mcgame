// Open-sourced by BaoLT

package item

import (
	"context"
	"strconv"

	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) ReplacePetEquipmentTemplate(ctx context.Context, charID int64, itemID int64, templateID int) (*domainitem.Item, error) {
	it, err := s.GetItemByID(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if it.ItemType != domainitem.ItemTypeEquipment || templateID <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}

	it.TemplateID = templateID
	if it.Properties != nil {
		for _, key := range []string{"tid", "cid", "giid", "tplId"} {
			if _, exists := it.Properties[key]; exists {
				it.Properties[key] = templateID
			}
		}
	}
	if err := s.itemRepo.Update(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

func (s *Service) SetPetEquipmentDisplayColor(ctx context.Context, charID int64, itemID int64, displayColor int) (*domainitem.Item, error) {
	it, err := s.GetItemByID(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if it.ItemType != domainitem.ItemTypeEquipment || displayColor < 0 || displayColor > 4 {
		return nil, pkgerrors.ErrInvalidInput
	}

	it.ColorCode = domainitem.EquipmentColorCodeFromDisplayColor(displayColor)
	if it.Properties == nil {
		it.Properties = make(map[string]interface{})
	}
	it.Properties["color"] = strconv.Itoa(displayColor)
	if err := s.itemRepo.Update(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

func (s *Service) SetPetEquipmentBindProps(ctx context.Context, charID int64, itemID int64, bindNum1 int, bindNum2 int) (*domainitem.Item, error) {
	it, err := s.GetItemByID(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if it.ItemType != domainitem.ItemTypeEquipment || bindNum1 < 0 || bindNum2 < 0 {
		return nil, pkgerrors.ErrInvalidInput
	}

	if it.Properties == nil {
		it.Properties = make(map[string]interface{})
	}
	it.Properties["bindMainPropNum1"] = bindNum1
	it.Properties["bindMainPropNum2"] = bindNum2
	if err := s.itemRepo.Update(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}
