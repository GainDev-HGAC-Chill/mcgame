// Cached guild-loop commands use the shared player mutation barrier and reconcile committed snapshots without dirty writes.
package cache

import (
	"context"

	domainquest "mcgame-server/internal/domain/quest"
)

type CachedQuestLoopRepository struct {
	domainquest.LoopRepository
	cache    *PlayerCache
	commands domainquest.GuildLoopCommandRepository
}

func NewCachedQuestLoopRepository(cache *PlayerCache, repository domainquest.LoopRepository) *CachedQuestLoopRepository {
	commands, _ := repository.(domainquest.GuildLoopCommandRepository)
	return &CachedQuestLoopRepository{LoopRepository: repository, cache: cache, commands: commands}
}

func (r *CachedQuestLoopRepository) CompleteGuildLoopAtomic(ctx context.Context, command domainquest.GuildLoopCompletionCommand) (*domainquest.GuildLoopCompletionResult, error) {
	if r.cache == nil || r.commands == nil {
		return nil, errStaleCachedMutation
	}
	var result *domainquest.GuildLoopCompletionResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.commands.CompleteGuildLoopAtomic(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		data.syncCommittedGuildLoop(command.ProgressID, result)
	})
	return result, err
}
