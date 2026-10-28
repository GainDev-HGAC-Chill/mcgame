// PostgreSQL guild building command persistence.
package postgres

import (
	"context"
	"fmt"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
)

func (r *GuildRepository) ListGuildBuildings(ctx context.Context, guildID int64) ([]*domainguild.BuildingState, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_guild_buildings($1)", guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild buildings: %w", err)
	}
	defer rows.Close()
	result := make([]*domainguild.BuildingState, 0)
	for rows.Next() {
		state := &domainguild.BuildingState{}
		if err := scanGuildBuilding(rows, state); err != nil {
			return nil, fmt.Errorf("failed to scan guild building: %w", err)
		}
		result = append(result, state)
	}
	return result, rows.Err()
}

func (r *GuildRepository) StartGuildBuildingAtomic(ctx context.Context, command appguild.BuildingStartCommand) (*domainguild.BuildingState, error) {
	state := &domainguild.BuildingState{}
	err := scanGuildBuilding(r.db.pool.QueryRow(ctx, "select * from player.start_guild_building_atomic($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		command.GuildID, command.ActorID, command.SourceID, command.ExtendID, command.TargetTemplateID, command.CurrentTemplateID,
		command.Mode, command.ExpectedRevision, command.PrerequisiteIDs, command.FamilyIDs, command.NumLimit), state)
	if err != nil {
		return nil, fmt.Errorf("failed to start guild building: %w", err)
	}
	return state, nil
}

func (r *GuildRepository) FinishGuildBuildingAtomic(ctx context.Context, command appguild.BuildingFinishCommand) (*domainguild.BuildingState, error) {
	state := &domainguild.BuildingState{}
	err := scanGuildBuilding(r.db.pool.QueryRow(ctx, "select * from player.finish_guild_building_atomic($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		command.GuildID, command.ActorID, command.BuildingID, command.TargetTemplateID, command.ExpectedRevision,
		command.MoneyCost, command.ExperienceCost, command.GeneralCost, command.RareCost, command.SpecialCost, command.HallLevel), state)
	if err != nil {
		return nil, fmt.Errorf("failed to finish guild building: %w", err)
	}
	return state, nil
}

func (r *GuildRepository) ResetGuildBuildingAtomic(ctx context.Context, guildID, actorID, buildingID int64) (*domainguild.BuildingState, error) {
	state := &domainguild.BuildingState{}
	if err := scanGuildBuilding(r.db.pool.QueryRow(ctx, "select * from player.reset_guild_building_atomic($1,$2,$3)", guildID, actorID, buildingID), state); err != nil {
		return nil, fmt.Errorf("failed to reset guild building: %w", err)
	}
	return state, nil
}

type guildBuildingScanner interface {
	Scan(...interface{}) error
}

func scanGuildBuilding(scanner guildBuildingScanner, state *domainguild.BuildingState) error {
	return scanner.Scan(&state.ID, &state.GuildID, &state.ExtendID, &state.TemplateID, &state.PendingTemplateID, &state.Mode, &state.ActorID, &state.Revision, &state.CreatedAt, &state.UpdatedAt)
}

var _ appguild.BuildingCommandRepository = (*GuildRepository)(nil)
