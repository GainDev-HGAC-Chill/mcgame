package item

import (
	"context"
	"math"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	silverGiftVoucherTemplateID = 654
	goldGiftVoucherTemplateID   = 1064
	goldCardTemplateID          = 3848
	openSilverGiftAmount        = int64(10000)
	goldGiftAmount              = int64(50)
)

var goldCardAmounts = [...]int64{88, 188, 288, 588, 888, 1888}

func isGiftCurrencyVoucher(templateID int) bool {
	return templateID == silverGiftVoucherTemplateID || templateID == goldGiftVoucherTemplateID || templateID == goldCardTemplateID
}

func (s *Service) useGiftCurrencyVoucher(ctx context.Context, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate, count, targetType int) (map[string]interface{}, error) {
	if tpl == nil || int(tpl.ID) != it.TemplateID || int(tpl.Type) != 550 || int(tpl.UseType) != 1 || (targetType != 0 && targetType != 1) {
		return nil, pkgerrors.ErrCannotUseItem
	}
	if char.Level < int(tpl.ReqLevel) {
		return nil, pkgerrors.ErrInsufficientLevel
	}
	if count <= 0 || count > it.StackCount {
		return nil, pkgerrors.ErrInvalidArgs
	}

	clientKey := "goldBind"
	current := char.GoldBind
	delta := int64(0)
	switch it.TemplateID {
	case silverGiftVoucherTemplateID:
		clientKey = "moneyBind"
		current = char.MoneyBind
		if int64(count) > math.MaxInt64/openSilverGiftAmount {
			return nil, pkgerrors.ErrInvalidArgs
		}
		delta = int64(count) * openSilverGiftAmount
	case goldGiftVoucherTemplateID:
		if int64(count) > math.MaxInt64/goldGiftAmount {
			return nil, pkgerrors.ErrInvalidArgs
		}
		delta = int64(count) * goldGiftAmount
	case goldCardTemplateID:
		for range count {
			amount := goldCardAmounts[s.randomIntInclusive(0, len(goldCardAmounts)-1)]
			if delta > math.MaxInt64-amount {
				return nil, pkgerrors.ErrInvalidInput
			}
			delta += amount
		}
	default:
		return nil, pkgerrors.ErrCannotUseItem
	}
	if current < 0 || current > math.MaxInt64-delta {
		return nil, pkgerrors.ErrInvalidInput
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
	if clientKey == "moneyBind" {
		char.MoneyBind = current + delta
	} else {
		char.GoldBind = current + delta
	}

	return map[string]interface{}{
		"granted_currencies": []GrantedCurrencyResult{{ClientKey: clientKey, Delta: delta, Total: float64(current + delta)}},
		"consumed":           count,
		"remaining":          remaining,
		"item_id":            it.ID,
		"used_item":          it,
	}, nil
}
