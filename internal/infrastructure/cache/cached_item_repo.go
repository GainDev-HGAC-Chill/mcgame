// Open-sourced by BaoLT

// CachedItemRepository wraps item.Repository with caching.
// Create writes to DB (needs ID), other writes update cache only.
// Delete tracks deleted IDs for batch persistence.
package cache

import (
	"context"
	"go.uber.org/zap"
	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
	"strings"
)

type CachedItemRepository struct {
	cache    *PlayerCache
	delegate item.Repository
}

func NewCachedItemRepository(cache *PlayerCache, delegate item.Repository) *CachedItemRepository {
	return &CachedItemRepository{cache: cache, delegate: delegate}
}

func (r *CachedItemRepository) CommitCharacterRename(ctx context.Context, characterID int64, card *item.Item, newName string, fact domainachievement.Fact) (int, error) {
	committer, ok := r.delegate.(interface {
		CommitCharacterRename(context.Context, int64, *item.Item, string, domainachievement.Fact) (int, error)
	})
	if !ok {
		return 0, pkgerrors.ErrSystemError
	}
	remaining := 0
	err := r.cache.runPlayerCommand(ctx, characterID, func() error {
		var commandErr error
		remaining, commandErr = committer.CommitCharacterRename(ctx, characterID, card, newName, fact)
		return commandErr
	}, func(data *PlayerData) {
		data.syncCommittedItemCount(card.ID, remaining)
		data.setCharacterNameWithoutDirty(newName)
	})
	if err != nil {
		return 0, err
	}
	if r.cache.warmLoader != nil {
		itemID := card.ID
		if remaining == 0 {
			itemID = 0
		}
		_ = r.cache.warmLoader.UpdateItem(ctx, characterID, itemID, card.SlotType, card.SlotIndex)
	}
	return remaining, nil
}

func (r *CachedItemRepository) FindByID(ctx context.Context, id int64) (*item.Item, error) {
	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if it := data.GetItem(id); it != nil {
			r.cache.mu.RUnlock()
			return it, nil
		}
	}
	r.cache.mu.RUnlock()
	return r.delegate.FindByID(ctx, id)
}

func (r *CachedItemRepository) FindByCharacterID(ctx context.Context, charID int64) ([]*item.Item, error) {
	if data := r.cache.GetPlayer(charID); data != nil {
		return data.GetItems(), nil
	}

	if r.cache.warmLoader != nil {
		itemMap, err := r.cache.warmLoader.GetItems(ctx, charID)
		if err == nil && itemMap != nil {
			items := make([]*item.Item, 0, len(itemMap))
			for _, it := range itemMap {
				items = append(items, it)
			}
			return items, nil
		}
	}

	return r.delegate.FindByCharacterID(ctx, charID)
}

func (r *CachedItemRepository) FindByCharacterAndSlotType(ctx context.Context, charID int64, slotType item.SlotType) ([]*item.Item, error) {
	if data := r.cache.GetPlayer(charID); data != nil {
		items := data.GetItems()
		var result []*item.Item
		for _, it := range items {
			if it.SlotType == slotType {
				result = append(result, it)
			}
		}
		return result, nil
	}

	if r.cache.warmLoader != nil {
		itemMap, err := r.cache.warmLoader.GetItems(ctx, charID)
		if err == nil && itemMap != nil {
			var result []*item.Item
			for _, it := range itemMap {
				if it.SlotType == slotType {
					result = append(result, it)
				}
			}
			return result, nil
		}
	}

	return r.delegate.FindByCharacterAndSlotType(ctx, charID, slotType)
}

func (r *CachedItemRepository) FindBySlot(ctx context.Context, charID int64, slotType item.SlotType, slotIndex int) (*item.Item, error) {
	if data := r.cache.GetPlayer(charID); data != nil {
		items := data.GetItems()
		for _, it := range items {
			if it.SlotType == slotType && it.SlotIndex == slotIndex {
				return it, nil
			}
		}
		return nil, pkgerrors.ErrItemNotFound
	}

	if r.cache.warmLoader != nil {
		itemMap, err := r.cache.warmLoader.GetItems(ctx, charID)
		if err == nil && itemMap != nil {
			for _, it := range itemMap {
				if it.SlotType == slotType && it.SlotIndex == slotIndex {
					return it, nil
				}
			}
			return nil, pkgerrors.ErrItemNotFound
		}
	}

	return r.delegate.FindBySlot(ctx, charID, slotType, slotIndex)
}

func (r *CachedItemRepository) FindEquipped(ctx context.Context, charID int64) ([]*item.Item, error) {
	return r.FindByCharacterAndSlotType(ctx, charID, item.SlotTypeEquipped)
}

func (r *CachedItemRepository) Create(ctx context.Context, it *item.Item) error {
	// First attempt: direct creation
	err := r.delegate.Create(ctx, it)
	if err == nil {
		if data := r.cache.GetPlayer(it.CharacterID); data != nil {
			data.AddItemWithoutDirty(it)
		}
		return nil
	}

	// Check if the error is a duplicate key violation (race condition)
	// Check if the error is a duplicate key violation (race condition)
	if isUniqueViolation(err) {
		// Force sync for this player: flush dirty items (like the one that just moved) to DB
		if saveErr := r.cache.SavePlayer(ctx, it.CharacterID); saveErr != nil {
			return err
		}

		// Retry Loop: Try up to 5 times to find a free slot
		for attempt := 0; attempt < 5; attempt++ {
			// AUTO-HEAL: If this is a Bag item, find a NEW empty slot using the updated cache
			if it.SlotType == item.SlotTypeBag {
				newSlot, slotErr := r.FindFirstEmptySlot(ctx, it.CharacterID, item.SlotTypeBag, item.DefaultBagSlots)
				// If we can't find a slot, we can't proceed
				if slotErr != nil {
					return slotErr
				}
				it.SlotIndex = newSlot
			}

			// Retry creation
			if retryErr := r.delegate.Create(ctx, it); retryErr == nil {
				// SUCCESS
				if data := r.cache.GetPlayer(it.CharacterID); data != nil {
					data.AddItemWithoutDirty(it)
				}
				return nil
			} else {
				// If error is NOT a unique violation, return immediately (real error)
				if !isUniqueViolation(retryErr) {
					return retryErr
				}
				// If it IS a unique violation, loop again to pick the next slot
			}
		}

		// If we ran out of attempts, return the original error (or the last one)
		return err
	}

	return err
}

func isUniqueViolation(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}

func shouldTrackWarmInventorySlot(slotType item.SlotType, slotIndex int) bool {
	capacity, tracked := warmSlotCapacity(slotType)
	if !tracked {
		return false
	}
	return slotIndex >= 0 && slotIndex < capacity
}

func (r *CachedItemRepository) Update(ctx context.Context, it *item.Item) error {
	if it.SlotType == item.SlotTypeEquipped {
		_, err := r.UpdateEquippedSlotsConditional(ctx, it.CharacterID, func(_ *character.Character, items []*item.Item) ([]*item.Item, error) {
			for _, current := range items {
				if current.ID == it.ID && current.SlotType == item.SlotTypeEquipped && current.SlotIndex == it.SlotIndex && current.TemplateID == it.TemplateID {
					return []*item.Item{it}, nil
				}
			}
			return nil, pkgerrors.ErrCannotEquip
		})
		return err
	}
	if data := r.cache.GetPlayer(it.CharacterID); data != nil {
		if !data.SetItemAtRevision(it) {
			return errStaleCachedMutation
		}
		if r.cache.warmLoader != nil && shouldTrackWarmInventorySlot(it.SlotType, it.SlotIndex) {
			_ = r.cache.warmLoader.UpdateItem(ctx, it.CharacterID, it.ID, it.SlotType, it.SlotIndex)
		}
		return nil
	}

	if err := r.delegate.Update(ctx, it); err != nil {
		return err
	}

	if r.cache.warmLoader != nil && shouldTrackWarmInventorySlot(it.SlotType, it.SlotIndex) {
		_ = r.cache.warmLoader.UpdateItem(ctx, it.CharacterID, it.ID, it.SlotType, it.SlotIndex)
	}
	return nil
}

func (r *CachedItemRepository) Delete(ctx context.Context, id int64) error {
	var foundInCache bool

	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if it := data.GetItem(id); it != nil {
			foundInCache = true
			if !data.DeleteItemAtRevision(id, it.CacheRevision) {
				r.cache.mu.RUnlock()
				return errStaleCachedMutation
			}
			if r.cache.warmLoader != nil {
				_ = r.cache.warmLoader.UpdateItem(ctx, it.CharacterID, 0, it.SlotType, it.SlotIndex)
			}
			break
		}
	}
	r.cache.mu.RUnlock()

	if foundInCache {
		return nil
	}
	return r.delegate.Delete(ctx, id)
}

func (r *CachedItemRepository) DeleteByCharacterID(ctx context.Context, charID int64) error {
	if r.cache.warmLoader != nil {
		_ = r.cache.warmLoader.DeleteFromRedis(ctx, charID)
	}
	return r.delegate.DeleteByCharacterID(ctx, charID)
}

func (r *CachedItemRepository) MoveItem(ctx context.Context, id int64, slotType item.SlotType, slotIndex int) error {
	if slotType == item.SlotTypeEquipped {
		return pkgerrors.ErrCannotEquip
	}
	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if it := data.GetItem(id); it != nil {
			r.cache.mu.RUnlock()
			it.SlotType = slotType
			it.SlotIndex = slotIndex
			if !data.SetItemAtRevision(it) {
				return errStaleCachedMutation
			}

			if r.cache.warmLoader != nil && shouldTrackWarmInventorySlot(slotType, slotIndex) {
				_ = r.cache.warmLoader.UpdateItem(ctx, it.CharacterID, id, slotType, slotIndex)
			}
			return nil
		}
	}
	r.cache.mu.RUnlock()

	if err := r.delegate.MoveItem(ctx, id, slotType, slotIndex); err != nil {
		return err
	}

	if r.cache.warmLoader != nil && shouldTrackWarmInventorySlot(slotType, slotIndex) {
		if it, err := r.delegate.FindByID(ctx, id); err == nil && it != nil {
			_ = r.cache.warmLoader.UpdateItem(ctx, it.CharacterID, id, slotType, slotIndex)
		}
	}

	return nil
}

func (r *CachedItemRepository) UpdateStack(ctx context.Context, id int64, stackCount int) error {
	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if it := data.GetItem(id); it != nil {
			r.cache.mu.RUnlock()
			it.StackCount = stackCount
			if it.SlotType == item.SlotTypeEquipped {
				return r.Update(ctx, it)
			}
			if !data.SetItemAtRevision(it) {
				return errStaleCachedMutation
			}
			return nil
		}
	}
	r.cache.mu.RUnlock()
	return r.delegate.UpdateStack(ctx, id, stackCount)
}

func (r *CachedItemRepository) FindFirstEmptySlot(ctx context.Context, charID int64, slotType item.SlotType, maxSlots int) (int, error) {
	if data := r.cache.GetPlayer(charID); data != nil {
		items := data.GetItems()
		occupied := make(map[int]bool)
		for _, it := range items {
			if it.SlotType == slotType {
				occupied[it.SlotIndex] = true
			}
		}
		for i := 0; i < maxSlots; i++ {
			if !occupied[i] {
				return i, nil
			}
		}
		return -1, pkgerrors.ErrInventoryFull
	}

	if r.cache.warmLoader != nil {
		itemMap, err := r.cache.warmLoader.GetItems(ctx, charID)
		if err == nil && itemMap != nil {
			occupied := make(map[int]bool)
			for _, it := range itemMap {
				if it.SlotType == slotType {
					occupied[it.SlotIndex] = true
				}
			}
			for i := 0; i < maxSlots; i++ {
				if !occupied[i] {
					return i, nil
				}
			}
			return -1, pkgerrors.ErrInventoryFull
		}
	}

	return r.delegate.FindFirstEmptySlot(ctx, charID, slotType, maxSlots)
}

func (r *CachedItemRepository) CountBySlotType(ctx context.Context, charID int64, slotType item.SlotType) (int, error) {
	if data := r.cache.GetPlayer(charID); data != nil {
		items := data.GetItems()
		count := 0
		for _, it := range items {
			if it.SlotType == slotType {
				count++
			}
		}
		return count, nil
	}

	if r.cache.warmLoader != nil {
		itemMap, err := r.cache.warmLoader.GetItems(ctx, charID)
		if err == nil && itemMap != nil {
			count := 0
			for _, it := range itemMap {
				if it.SlotType == slotType {
					count++
				}
			}
			return count, nil
		}
	}

	return r.delegate.CountBySlotType(ctx, charID, slotType)
}

// Đẩy các thay đổi đang chờ xuống database trước
// khi thực hiện transaction trực tiếp bằng PostgreSQL.
func (r *CachedItemRepository) FlushBeforeBloodBagRefill(
	ctx context.Context,
	charID int64,
) error {
	return r.cache.SavePlayer(ctx, charID)
}

// Đồng bộ cache sau khi giao dịch nạp túi cứu thương
// đã commit thành công.
func (r *CachedItemRepository) SyncCommittedBloodBagItem(
	ctx context.Context,
	it *item.Item,
	remaining int,
) {
	if it == nil {
		return
	}

	if data := r.cache.GetPlayer(it.CharacterID); data != nil {
		data.SyncCommittedItemCount(it.ID, remaining)

		if err := r.cache.saveToRedis(ctx, data); err != nil {
			r.cache.logger.Warn(
				"Failed to sync blood bag refill item to Redis",
				zap.Int64("character_id", it.CharacterID),
				zap.Int64("item_id", it.ID),
				zap.Error(err),
			)
		}
	}

	if r.cache.warmLoader != nil &&
		shouldTrackWarmInventorySlot(it.SlotType, it.SlotIndex) {

		itemID := it.ID
		if remaining <= 0 {
			itemID = 0
		}

		_ = r.cache.warmLoader.UpdateItem(
			ctx,
			it.CharacterID,
			itemID,
			it.SlotType,
			it.SlotIndex,
		)
	}
}
