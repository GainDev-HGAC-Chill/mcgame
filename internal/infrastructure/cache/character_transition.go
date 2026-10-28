// Conditional character transitions keep membership and current inventory locked until commit.
// Callbacks may inspect cloned entities but must not call repositories or reenter the cache.
package cache

import (
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
)

func (c *PlayerCache) UpdateCachedCharacterConditional(
	charID int64,
	transition func(*character.Character, []*item.Item) (bool, error),
) (*character.Character, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data := c.activeSessions[charID]
	if data == nil {
		return nil, false, nil
	}
	data.mutationMu.Lock()
	defer data.mutationMu.Unlock()
	data.mu.Lock()
	defer data.mu.Unlock()
	if data.Character == nil {
		return nil, false, nil
	}
	updated := cloneCharacter(data.Character)
	items := make([]*item.Item, 0, len(data.Items))
	for _, it := range data.Items {
		items = append(items, cloneItem(it))
	}
	changed, err := transition(updated, items)
	if err != nil || !changed {
		return cloneCharacter(data.Character), false, err
	}
	data.Character = cloneCharacter(updated)
	data.characterRevision++
	data.Character.CacheRevision = data.characterRevision
	data.revision++
	data.dirtyChar = true
	return updated, true, nil
}
