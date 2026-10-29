// Cached farm commands serialize database-owned mutations with player write-behind state.
package cache

import (
	"context"

	domainfarm "mcgame-server/internal/domain/farm"
)

type CachedFarmRepository struct {
	domainfarm.Repository
	cache    *PlayerCache
	commands domainfarm.CommandRepository
}

func NewCachedFarmRepository(cache *PlayerCache, repository domainfarm.Repository) *CachedFarmRepository {
	commands, _ := repository.(domainfarm.CommandRepository)
	return &CachedFarmRepository{Repository: repository, cache: cache, commands: commands}
}

func (r *CachedFarmRepository) PlantAtomic(ctx context.Context, command domainfarm.PlantCommand) (*domainfarm.PlantCommandResult, error) {
	if r.cache == nil || r.commands == nil {
		return nil, errStaleCachedMutation
	}
	var result *domainfarm.PlantCommandResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.commands.PlantAtomic(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		if result != nil {
			data.syncCommittedFarmState(result.Character, result.Seed)
		}
	})
	return result, err
}

func (r *CachedFarmRepository) HarvestAtomic(ctx context.Context, command domainfarm.HarvestCommand) (*domainfarm.HarvestCommandResult, error) {
	if r.cache == nil || r.commands == nil {
		return nil, errStaleCachedMutation
	}
	var result *domainfarm.HarvestCommandResult
	err := r.cache.runPlayerCommand(ctx, command.HarvesterCharacterID, func() error {
		var commandErr error
		result, commandErr = r.commands.HarvestAtomic(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		if result != nil {
			data.syncCommittedFarmState(nil, result.Reward)
		}
	})
	return result, err
}

func (r *CachedFarmRepository) RipenAndHarvestAtomic(ctx context.Context, command domainfarm.RipenAndHarvestCommand) (*domainfarm.RipenAndHarvestCommandResult, error) {
	if r.cache == nil || r.commands == nil {
		return nil, errStaleCachedMutation
	}
	var result *domainfarm.RipenAndHarvestCommandResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.commands.RipenAndHarvestAtomic(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		if result != nil {
			data.syncCommittedFarmState(result.Character, result.Reward)
		}
	})
	return result, err
}

func (r *CachedFarmRepository) UprootAtomic(ctx context.Context, command domainfarm.UprootCommand) (*domainfarm.UprootCommandResult, error) {
	if r.cache == nil || r.commands == nil {
		return nil, errStaleCachedMutation
	}
	var result *domainfarm.UprootCommandResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.commands.UprootAtomic(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		if result != nil {
			data.syncCommittedFarmState(result.Character)
		}
	})
	return result, err
}

var _ domainfarm.Repository = (*CachedFarmRepository)(nil)
var _ domainfarm.CommandRepository = (*CachedFarmRepository)(nil)
