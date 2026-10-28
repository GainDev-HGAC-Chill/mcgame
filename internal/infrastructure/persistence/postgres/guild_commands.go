// PostgreSQL-backed atomic guild economy commands.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func (r *GuildRepository) CreateGuildAtomic(ctx context.Context, command appguild.CreateGuildCommand) (*appguild.CreateGuildResult, error) {
	ranks, err := json.Marshal(command.Ranks)
	if err != nil {
		return nil, fmt.Errorf("failed to encode guild rank settings: %w", err)
	}
	result := &appguild.CreateGuildResult{}
	result.Leader.CharacterID = command.LeaderID
	var guildID int64
	err = r.db.pool.QueryRow(ctx, "select * from player.create_guild_atomic($1, $2, $3, $4)", command.LeaderID, command.Name, command.Cost, ranks).Scan(
		&guildID, &result.GuildName, &result.Leader.Money, &result.Leader.MoneyBind, &result.Leader.Gold,
		&result.Leader.GuildContrib, &result.Leader.DonateContrib, &result.Leader.GuildRestoreNormal, &result.Leader.GuildRestoreDonate,
	)
	if err != nil {
		return nil, mapGuildCommandError("create guild", err)
	}
	result.GuildID = guildID
	result.Leader.GuildID = &guildID
	return result, nil
}

func (r *GuildRepository) ContributeMoneyAtomic(ctx context.Context, command appguild.ContributeMoneyCommand) (*appguild.ContributeMoneyResult, error) {
	result := &appguild.ContributeMoneyResult{}
	result.Character.CharacterID = command.CharacterID
	result.Character.GuildID = &command.GuildID
	err := r.db.pool.QueryRow(ctx, "select * from player.contribute_guild_money_atomic($1, $2, $3, $4)",
		command.GuildID, command.CharacterID, command.Silver, command.Gold,
	).Scan(
		&result.CharacterName, &result.Character.Money, &result.Character.MoneyBind, &result.Character.Gold,
		&result.Character.GuildContrib, &result.Character.DonateContrib, &result.Character.GuildRestoreNormal, &result.Character.GuildRestoreDonate,
		&result.GuildFunds, &result.MemberDonate, &result.MemberTotal, &result.MemberWeekly,
	)
	if err != nil {
		return nil, mapGuildCommandError("contribute guild money", err)
	}
	return result, nil
}

func (r *GuildRepository) ContributeMaterialAtomic(ctx context.Context, command appguild.ContributeMaterialCommand) (*appguild.ContributeMaterialResult, error) {
	result := &appguild.ContributeMaterialResult{}
	result.Character.CharacterID = command.CharacterID
	result.Character.GuildID = &command.GuildID
	result.ItemID = command.ItemID
	result.WarehouseSlot = &domainguild.GuildWarehouseSlot{GuildID: command.GuildID, TemplateID: command.WarehouseTemplateID}
	err := r.db.pool.QueryRow(ctx, "select * from player.contribute_guild_material_atomic($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		command.GuildID, command.CharacterID, command.ItemID, command.ItemTemplateID, command.ItemCount,
		command.WarehouseTemplateID, command.WarehouseCount, command.WarehouseMaxStack, command.WarehousePageCount, command.DonateGain,
	).Scan(
		&result.CharacterName, &result.ItemRemaining, &result.WarehouseSlot.ID, &result.WarehouseSlot.SlotIndex,
		&result.WarehouseSlot.StackCount, &result.Character.Money, &result.Character.MoneyBind, &result.Character.Gold,
		&result.Character.GuildContrib, &result.Character.DonateContrib, &result.Character.GuildRestoreNormal, &result.Character.GuildRestoreDonate,
		&result.MemberDonate, &result.MemberTotal, &result.MemberWeekly,
	)
	if err != nil {
		return nil, mapGuildCommandError("contribute guild material", err)
	}
	return result, nil
}

var _ appguild.CommandRepository = (*GuildRepository)(nil)

func mapGuildCommandError(action string, err error) error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Message {
		case "GUILD_NAME_TAKEN":
			return appguild.ErrGuildNameTaken
		case "ALREADY_IN_GUILD":
			return appguild.ErrAlreadyInGuild
		case "GUILD_NOT_FOUND", "GUILD_APPLICATION_NOT_FOUND":
			return appguild.ErrGuildNotFound
		case "NOT_IN_GUILD":
			return appguild.ErrNotInGuild
		case "GUILD_FULL":
			return appguild.ErrGuildFull
		case "CANNOT_KICK_LEADER":
			return appguild.ErrCannotKickLeader
		case "GUILD_UNAUTHORIZED", "GUILD_EXIT_UNAUTHORIZED":
			return pkgerrors.ErrUnauthorized
		case "GUILD_APPLICATION_REJECTED":
			return pkgerrors.ErrUnauthorized
		case "INVALID_GUILD_APPLICATION":
			return appguild.ErrAlreadyInGuild
		case "MEMBERSHIP_DRIFT", "INVALID_GUILD_APPLICATION_STATE", "INVALID_GUILD_MEMBER_STATE", "GUILD_RESTORE_OVERFLOW":
			return appguild.ErrUnsupportedAction
		case "INSUFFICIENT_FUNDS":
			return appguild.ErrInsufficientFunds
		case "INSUFFICIENT_LEVEL":
			return appguild.ErrInsufficientLevel
		case "INVALID_CONTRIBUTION":
			return appguild.ErrInvalidContribution
		case "GUILD_WAREHOUSE_FULL":
			return pkgerrors.ErrInventoryFull
		}
	}
	return fmt.Errorf("failed to %s atomically: %w", action, err)
}
