// Player mutation barriers serialize write-behind persistence and database-owned commands.
package cache

import (
	"context"
	"errors"
	"slices"
	"sort"
	"sync"

	"go.uber.org/zap"
)

var errStaleCachedMutation = errors.New("stale cached player mutation")

func (c *PlayerCache) runPlayerCommand(ctx context.Context, characterID int64, command func() error, reconcile func(*PlayerData)) error {
	commandLock := c.playerCommandLock(characterID)
	commandLock.Lock()
	defer commandLock.Unlock()
	data := c.GetPlayer(characterID)
	if data == nil {
		return command()
	}
	data.mutationMu.Lock()
	defer data.mutationMu.Unlock()
	if data.IsDirty() {
		if err := c.persistPlayerDataExclusive(ctx, data); err != nil {
			return err
		}
	}
	if err := command(); err != nil {
		return err
	}
	if reconcile != nil {
		reconcile(data)
	}
	if err := c.saveToRedis(ctx, data); err != nil {
		c.logger.Warn("Failed to reconcile player command to Redis", zap.Int64("character_id", characterID), zap.Error(err))
	}
	return nil
}

func (c *PlayerCache) runPlayersCommand(ctx context.Context, characterIDs []int64, command func() error, reconcile func(int64, *PlayerData)) error {
	ids := append([]int64(nil), characterIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	ids = slices.Compact(ids)
	locks := make([]*sync.Mutex, 0, len(ids))
	dataRows := make([]*PlayerData, 0, len(ids))
	for _, characterID := range ids {
		lock := c.playerCommandLock(characterID)
		lock.Lock()
		locks = append(locks, lock)
	}
	defer func() {
		for index := len(locks) - 1; index >= 0; index-- {
			locks[index].Unlock()
		}
	}()
	for _, characterID := range ids {
		if data := c.GetPlayer(characterID); data != nil {
			data.mutationMu.Lock()
			dataRows = append(dataRows, data)
		}
	}
	defer func() {
		for index := len(dataRows) - 1; index >= 0; index-- {
			dataRows[index].mutationMu.Unlock()
		}
	}()
	for _, data := range dataRows {
		if data.IsDirty() {
			if err := c.persistPlayerDataExclusive(ctx, data); err != nil {
				return err
			}
		}
	}
	if err := command(); err != nil {
		return err
	}
	for _, data := range dataRows {
		if reconcile != nil {
			reconcile(data.CharacterID, data)
		}
		if err := c.saveToRedis(ctx, data); err != nil {
			c.logger.Warn("Failed to reconcile multi-player command to Redis", zap.Int64("character_id", data.CharacterID), zap.Error(err))
		}
	}
	return nil
}
