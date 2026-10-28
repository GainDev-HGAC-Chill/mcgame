// Equipped-slot transitions serialize eligibility and all inventory moves with level changes.
// Callbacks inspect detached current state and must not call repositories or reenter the cache.
package cache

import (
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

func (c *PlayerCache) UpdateEquippedSlotsConditional(charID int64, transition func(*character.Character, []*item.Item) ([]*item.Item, error)) (*character.Character, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data := c.activeSessions[charID]
	if data == nil {
		return nil, pkgerrors.ErrCharacterNotFound
	}
	data.mutationMu.Lock()
	defer data.mutationMu.Unlock()
	data.mu.Lock()
	defer data.mu.Unlock()
	if data.Character == nil {
		return nil, pkgerrors.ErrCharacterNotFound
	}
	current := cloneCharacter(data.Character)
	items := make([]*item.Item, 0, len(data.Items))
	for _, it := range data.Items {
		items = append(items, cloneItem(it))
	}
	changes, err := transition(current, items)
	if err != nil {
		return nil, err
	}
	for _, it := range changes {
		if it == nil || data.Items[it.ID] == nil || it.CharacterID != charID {
			return nil, pkgerrors.ErrItemNotOwned
		}
	}
	if len(changes) > 0 {
		for _, it := range changes {
			data.Items[it.ID] = cloneItem(it)
			data.itemRevisions[it.ID] = data.itemRevision(it.ID) + 1
			data.Items[it.ID].CacheRevision = data.itemRevisions[it.ID]
			data.dirtyItems[it.ID] = true
		}
		data.revision++
	}
	return cloneCharacter(data.Character), nil
}
