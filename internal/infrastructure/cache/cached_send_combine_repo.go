// Cached Send Combine commands flush write-behind state and reconcile committed purchases.
package cache

import (
	"context"

	domainactivity "mcgame-server/internal/domain/activity"
)

type CachedSendCombineRepository struct {
	cache    *PlayerCache
	delegate domainactivity.SendCombineRepository
}

func NewCachedSendCombineRepository(cache *PlayerCache, delegate domainactivity.SendCombineRepository) *CachedSendCombineRepository {
	return &CachedSendCombineRepository{cache: cache, delegate: delegate}
}

func (r *CachedSendCombineRepository) PurchaseSendCombine(ctx context.Context, command domainactivity.SendCombinePurchaseCommand) (*domainactivity.SendCombinePurchaseResult, error) {
	var result *domainactivity.SendCombinePurchaseResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.delegate.PurchaseSendCombine(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		data.syncCommittedSendCombine(result)
	})
	if err != nil {
		return nil, err
	}
	if r.cache.warmLoader != nil {
		for _, committed := range result.Items {
			_ = r.cache.warmLoader.UpdateItem(ctx, command.CharacterID, committed.ID, committed.SlotType, committed.SlotIndex)
		}
	}
	return result, nil
}

func (r *CachedSendCombineRepository) ListSendCombineBuyCounts(ctx context.Context, characterID int64, eventEndMs int64) (map[string]int, error) {
	return r.delegate.ListSendCombineBuyCounts(ctx, characterID, eventEndMs)
}

var _ domainactivity.SendCombineRepository = (*CachedSendCombineRepository)(nil)
