// Cached guild commands flush player write-behind state before DB commit and reconcile committed snapshots afterward.
package cache

import (
	"context"
	"fmt"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
)

type CachedGuildRepository struct {
	appguild.Repository
	cache    *PlayerCache
	commands appguild.CommandRepository
}

func NewCachedGuildRepository(cache *PlayerCache, repository appguild.Repository) *CachedGuildRepository {
	commands, _ := repository.(appguild.CommandRepository)
	return &CachedGuildRepository{Repository: repository, cache: cache, commands: commands}
}

func (r *CachedGuildRepository) CreateGuildAtomic(ctx context.Context, command appguild.CreateGuildCommand) (*appguild.CreateGuildResult, error) {
	if !r.guildCommandsSafe() {
		return nil, appguild.ErrUnsupportedAction
	}
	var result *appguild.CreateGuildResult
	err := r.cache.runPlayerCommand(ctx, command.LeaderID, func() error {
		var err error
		result, err = r.commands.CreateGuildAtomic(ctx, command)
		return err
	}, func(data *PlayerData) {
		data.syncCommittedGuildState(result.Leader.GuildID, result.Leader.Money, result.Leader.MoneyBind, result.Leader.Gold, result.Leader.GuildContrib, result.Leader.DonateContrib, result.Leader.GuildRestoreNormal, result.Leader.GuildRestoreDonate)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *CachedGuildRepository) ContributeMoneyAtomic(ctx context.Context, command appguild.ContributeMoneyCommand) (*appguild.ContributeMoneyResult, error) {
	if !r.guildCommandsSafe() {
		return nil, appguild.ErrUnsupportedAction
	}
	var result *appguild.ContributeMoneyResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var err error
		result, err = r.commands.ContributeMoneyAtomic(ctx, command)
		return err
	}, func(data *PlayerData) {
		data.syncCommittedGuildState(result.Character.GuildID, result.Character.Money, result.Character.MoneyBind, result.Character.Gold, result.Character.GuildContrib, result.Character.DonateContrib, result.Character.GuildRestoreNormal, result.Character.GuildRestoreDonate)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *CachedGuildRepository) ContributeMaterialAtomic(ctx context.Context, command appguild.ContributeMaterialCommand) (*appguild.ContributeMaterialResult, error) {
	if !r.guildCommandsSafe() {
		return nil, appguild.ErrUnsupportedAction
	}
	var result *appguild.ContributeMaterialResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var err error
		result, err = r.commands.ContributeMaterialAtomic(ctx, command)
		return err
	}, func(data *PlayerData) {
		data.syncCommittedGuildState(result.Character.GuildID, result.Character.Money, result.Character.MoneyBind, result.Character.Gold, result.Character.GuildContrib, result.Character.DonateContrib, result.Character.GuildRestoreNormal, result.Character.GuildRestoreDonate)
		data.syncCommittedItemCount(result.ItemID, result.ItemRemaining)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *CachedGuildRepository) guildCommandsSafe() bool {
	return r.cache != nil && r.commands != nil
}

func (r *CachedGuildRepository) DevelopGuildSkillAtomic(ctx context.Context, guildID, actorID int64, development appguild.GuildSkillDevelopment) error {
	commands, ok := r.Repository.(appguild.GuildSkillCommandRepository)
	if !ok {
		return appguild.ErrUnsupportedAction
	}
	return commands.DevelopGuildSkillAtomic(ctx, guildID, actorID, development)
}

func (r *CachedGuildRepository) SetGuildName(ctx context.Context, guildID, actorID int64, name string) error {
	commands, ok := r.Repository.(appguild.GuildMetadataRepository)
	if !ok {
		return appguild.ErrUnsupportedAction
	}
	return commands.SetGuildName(ctx, guildID, actorID, name)
}

func (r *CachedGuildRepository) SetGuildNotice(ctx context.Context, guildID, actorID int64, announcement, description string) error {
	commands, ok := r.Repository.(appguild.GuildMetadataRepository)
	if !ok {
		return appguild.ErrUnsupportedAction
	}
	return commands.SetGuildNotice(ctx, guildID, actorID, announcement, description)
}

func (r *CachedGuildRepository) MoveGuildWarehouseAtomic(ctx context.Context, command appguild.WarehouseMoveCommand) (*appguild.WarehouseMoveResult, error) {
	commands, ok := r.Repository.(appguild.WarehouseCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.MoveGuildWarehouseAtomic(ctx, command)
}

func (r *CachedGuildRepository) DropGuildWarehouseAtomic(ctx context.Context, command appguild.WarehouseDropCommand) (*appguild.WarehouseDropResult, error) {
	commands, ok := r.Repository.(appguild.WarehouseCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.DropGuildWarehouseAtomic(ctx, command)
}

func (r *CachedGuildRepository) ListGuildBuildings(ctx context.Context, guildID int64) ([]*domainguild.BuildingState, error) {
	commands, ok := r.Repository.(appguild.BuildingCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.ListGuildBuildings(ctx, guildID)
}

func (r *CachedGuildRepository) StartGuildBuildingAtomic(ctx context.Context, command appguild.BuildingStartCommand) (*domainguild.BuildingState, error) {
	commands, ok := r.Repository.(appguild.BuildingCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.StartGuildBuildingAtomic(ctx, command)
}

func (r *CachedGuildRepository) FinishGuildBuildingAtomic(ctx context.Context, command appguild.BuildingFinishCommand) (*domainguild.BuildingState, error) {
	commands, ok := r.Repository.(appguild.BuildingCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.FinishGuildBuildingAtomic(ctx, command)
}

func (r *CachedGuildRepository) ResetGuildBuildingAtomic(ctx context.Context, guildID, actorID, buildingID int64) (*domainguild.BuildingState, error) {
	commands, ok := r.Repository.(appguild.BuildingCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.ResetGuildBuildingAtomic(ctx, guildID, actorID, buildingID)
}

func (r *CachedGuildRepository) ApplyGuildAtomic(ctx context.Context, guildID, characterID int64) (*domainguild.GuildApplication, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.ApplyGuildAtomic(ctx, guildID, characterID)
}

func (r *CachedGuildRepository) ApproveGuildApplicationAtomic(ctx context.Context, applicationID, approverID int64) (*appguild.ApproveApplicationResult, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok || r.cache == nil {
		return nil, appguild.ErrUnsupportedAction
	}
	application, err := r.Repository.GetApplicationByID(ctx, applicationID)
	if err != nil || application == nil {
		return nil, err
	}
	var result *appguild.ApproveApplicationResult
	err = r.cache.runPlayerCommand(ctx, application.CharacterID, func() error {
		var commandErr error
		result, commandErr = commands.ApproveGuildApplicationAtomic(ctx, applicationID, approverID)
		return commandErr
	}, func(data *PlayerData) {
		data.syncCommittedGuildState(result.Character.GuildID, result.Character.Money, result.Character.MoneyBind, result.Character.Gold, result.Character.GuildContrib, result.Character.DonateContrib, result.Character.GuildRestoreNormal, result.Character.GuildRestoreDonate)
	})
	return result, err
}

func (r *CachedGuildRepository) RefuseGuildApplicationAtomic(ctx context.Context, applicationID, approverID int64) (*domainguild.GuildApplication, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.RefuseGuildApplicationAtomic(ctx, applicationID, approverID)
}

func (r *CachedGuildRepository) CancelGuildApplicationAtomic(ctx context.Context, applicationID, characterID int64) (*domainguild.GuildApplication, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.CancelGuildApplicationAtomic(ctx, applicationID, characterID)
}

func (r *CachedGuildRepository) LeaveGuildAtomic(ctx context.Context, guildID, characterID int64) (*appguild.ExitGuildResult, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok || r.cache == nil {
		return nil, appguild.ErrUnsupportedAction
	}
	return r.runGuildExitCommand(ctx, characterID, func() (*appguild.ExitGuildResult, error) {
		return commands.LeaveGuildAtomic(ctx, guildID, characterID)
	})
}

func (r *CachedGuildRepository) KickGuildMemberAtomic(ctx context.Context, guildID, actorID, targetID int64) (*appguild.ExitGuildResult, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok || r.cache == nil {
		return nil, appguild.ErrUnsupportedAction
	}
	return r.runGuildExitCommand(ctx, targetID, func() (*appguild.ExitGuildResult, error) {
		return commands.KickGuildMemberAtomic(ctx, guildID, actorID, targetID)
	})
}

func (r *CachedGuildRepository) runGuildExitCommand(ctx context.Context, characterID int64, command func() (*appguild.ExitGuildResult, error)) (*appguild.ExitGuildResult, error) {
	var result *appguild.ExitGuildResult
	err := r.cache.runPlayerCommand(ctx, characterID, func() error {
		var commandErr error
		result, commandErr = command()
		return commandErr
	}, func(data *PlayerData) {
		data.syncCommittedGuildState(result.Character.GuildID, result.Character.Money, result.Character.MoneyBind, result.Character.Gold, result.Character.GuildContrib, result.Character.DonateContrib, result.Character.GuildRestoreNormal, result.Character.GuildRestoreDonate)
	})
	return result, err
}

func (r *CachedGuildRepository) TransferGuildLeadershipAtomic(ctx context.Context, guildID, oldLeaderID, newLeaderID int64) (*appguild.TransferLeadershipResult, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok {
		return nil, appguild.ErrUnsupportedAction
	}
	return commands.TransferGuildLeadershipAtomic(ctx, guildID, oldLeaderID, newLeaderID)
}

func (r *CachedGuildRepository) SetGuildMemberRankAtomic(ctx context.Context, guildID, actorID, targetID int64, rank int) error {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok {
		return appguild.ErrUnsupportedAction
	}
	return commands.SetGuildMemberRankAtomic(ctx, guildID, actorID, targetID, rank)
}

func (r *CachedGuildRepository) DisbandGuildAtomic(ctx context.Context, command appguild.DisbandGuildCommand) (*appguild.DisbandGuildResult, error) {
	commands, ok := r.Repository.(appguild.LifecycleCommandRepository)
	if !ok || r.cache == nil || len(command.ExpectedMemberIDs) == 0 {
		return nil, appguild.ErrUnsupportedAction
	}
	var result *appguild.DisbandGuildResult
	committed := make(map[int64]appguild.CharacterCommitState, len(command.ExpectedMemberIDs))
	err := r.cache.runPlayersCommand(ctx, command.ExpectedMemberIDs, func() error {
		var commandErr error
		result, commandErr = commands.DisbandGuildAtomic(ctx, command)
		if commandErr != nil {
			return commandErr
		}
		for _, state := range result.Characters {
			committed[state.CharacterID] = state
		}
		if len(committed) != len(command.ExpectedMemberIDs) {
			return fmt.Errorf("guild disband committed character mismatch")
		}
		for _, characterID := range command.ExpectedMemberIDs {
			if _, exists := committed[characterID]; !exists {
				return fmt.Errorf("guild disband missing committed character %d", characterID)
			}
		}
		return nil
	}, func(characterID int64, data *PlayerData) {
		state := committed[characterID]
		data.syncCommittedGuildState(state.GuildID, state.Money, state.MoneyBind, state.Gold, state.GuildContrib, state.DonateContrib, state.GuildRestoreNormal, state.GuildRestoreDonate)
	})
	return result, err
}

var _ appguild.Repository = (*CachedGuildRepository)(nil)
var _ appguild.CommandRepository = (*CachedGuildRepository)(nil)
