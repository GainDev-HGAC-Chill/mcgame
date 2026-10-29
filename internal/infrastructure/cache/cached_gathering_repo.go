// Cached gathering commands flush write-behind state and reconcile committed snapshots.
package cache

import (
	"context"

	domaingathering "mcgame-server/internal/domain/gathering"
)

type CachedGatheringRepository struct {
	cache    *PlayerCache
	delegate domaingathering.CommandRepository
}

func NewCachedGatheringRepository(cache *PlayerCache, delegate domaingathering.CommandRepository) *CachedGatheringRepository {
	return &CachedGatheringRepository{cache: cache, delegate: delegate}
}

func (r *CachedGatheringRepository) CompleteGathering(ctx context.Context, command domaingathering.CompletionCommand) (*domaingathering.CompletionResult, error) {
	if r.cache == nil || r.delegate == nil {
		return nil, errStaleCachedMutation
	}
	var result *domaingathering.CompletionResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.delegate.CompleteGathering(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		data.syncCommittedGathering(result)
	})
	return result, err
}

var _ domaingathering.CommandRepository = (*CachedGatheringRepository)(nil)
