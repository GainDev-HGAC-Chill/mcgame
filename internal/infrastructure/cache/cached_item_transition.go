// Item application operations use registered current cache state without repository calls under player locks.
package cache

import (
	"context"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
)

func (r *CachedItemRepository) UpdateEquippedSlotsConditional(ctx context.Context, charID int64, transition func(*character.Character, []*item.Item) ([]*item.Item, error)) (*character.Character, error) {
	var old, changed []*item.Item
	updated, err := r.cache.UpdateEquippedSlotsConditional(charID, func(char *character.Character, items []*item.Item) ([]*item.Item, error) {
		for _, it := range items {
			old = append(old, cloneItem(it))
		}
		var err error
		changed, err = transition(char, items)
		return changed, err
	})
	if err != nil {
		return nil, err
	}
	if r.cache.warmLoader != nil {
		ids := make(map[int64]bool, len(changed))
		for _, it := range changed {
			ids[it.ID] = true
		}
		for _, it := range old {
			if ids[it.ID] && shouldTrackWarmInventorySlot(it.SlotType, it.SlotIndex) {
				_ = r.cache.warmLoader.UpdateItem(ctx, charID, 0, it.SlotType, it.SlotIndex)
			}
		}
		for _, it := range changed {
			if shouldTrackWarmInventorySlot(it.SlotType, it.SlotIndex) {
				_ = r.cache.warmLoader.UpdateItem(ctx, charID, it.ID, it.SlotType, it.SlotIndex)
			}
		}
	}
	return updated, nil
}

func (r *CachedItemRepository) UpdateCachedCharacterConditional(charID int64, transition func(*character.Character, []*item.Item) (bool, error)) (*character.Character, bool, error) {
	return r.cache.UpdateCachedCharacterConditional(charID, transition)
}
