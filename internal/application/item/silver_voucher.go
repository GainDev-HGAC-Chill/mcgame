package item

import (
	"context"
	"math"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) useSilverVoucher(ctx context.Context, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate, count, targetType int) (map[string]interface{}, error) {
	if tpl == nil || tpl.ID != 1132 || int(tpl.Type) != 550 || int(tpl.UseType) != 1 || (targetType != 0 && targetType != 1) {
		return nil, pkgerrors.ErrCannotUseItem
	}
	if char.Level < int(tpl.ReqLevel) {
		return nil, pkgerrors.ErrInsufficientLevel
	}
	if count <= 0 || count > it.StackCount || int64(count) > math.MaxInt64/100000 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	delta := int64(count) * 100000
	key, current := "money", char.Money
	if it.IsBound {
		key, current = "moneyBind", char.MoneyBind
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
	if it.IsBound {
		char.MoneyBind = current + delta
	} else {
		char.Money = current + delta
	}
	return map[string]interface{}{
		"granted_currencies": []GrantedCurrencyResult{{ClientKey: key, Delta: delta, Total: float64(current + delta)}},
		"consumed":           count, "remaining": remaining, "item_id": it.ID, "used_item": it,
	}, nil
}
