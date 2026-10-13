package item

import (
	"context"
	"errors"
	"time"

	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

const reusableItemExpiresAtKey = "reusableExpiresAt"

func (s *Service) UsePreferredTimedOrConsumableItem(ctx context.Context, charID int64, templateIDs []int, now time.Time) (*domainitem.Item, int, bool, error) {
	for _, templateID := range templateIDs {
		it, err := s.GetItemByTemplateID(ctx, charID, templateID)
		if err != nil {
			if errors.Is(err, pkgerrors.ErrItemNotFound) {
				continue
			}
			return nil, 0, false, err
		}
		tpl := s.GetItemTemplate(templateID)
		if tpl == nil || tpl.T <= 0 {
			used, deleted, err := s.ConsumeItemStackByID(ctx, charID, it.ID, 1)
			return used, templateID, deleted, err
		}
		expiresAt, active := reusableItemExpiry(it)
		if active && !now.Before(expiresAt) {
			continue
		}
		if !active {
			if it.Properties == nil {
				it.Properties = make(map[string]interface{})
			}
			it.Properties[reusableItemExpiresAtKey] = now.Add(time.Duration(tpl.T) * time.Minute).UTC().Format(time.RFC3339Nano)
			if err := s.itemRepo.Update(ctx, it); err != nil {
				return nil, 0, false, err
			}
		}
		return it, templateID, false, nil
	}
	return nil, 0, false, nil
}

func reusableItemExpiry(it *domainitem.Item) (time.Time, bool) {
	if it == nil || it.Properties == nil {
		return time.Time{}, false
	}
	raw, ok := it.Properties[reusableItemExpiresAtKey].(string)
	if !ok || raw == "" {
		return time.Time{}, false
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, raw)
	return expiresAt, err == nil
}
