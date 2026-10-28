// PostgreSQL atomic guild warehouse operations.
package postgres

import (
	"context"
	"fmt"

	appguild "mcgame-server/internal/application/guild"
)

func (r *GuildRepository) MoveGuildWarehouseAtomic(ctx context.Context, cmd appguild.WarehouseMoveCommand) (*appguild.WarehouseMoveResult, error) {
	result := &appguild.WarehouseMoveResult{}
	err := r.db.pool.QueryRow(ctx, "select * from player.move_guild_warehouse_atomic($1,$2,$3,$4,$5,$6,$7,$8)",
		cmd.GuildID, cmd.ActorID, cmd.SourceID, cmd.SourceIndex, cmd.TargetIndex, cmd.TemplateID, cmd.MaxStack, cmd.PageCount,
	).Scan(&result.SourceID, &result.TargetID, &result.SourceCount, &result.TargetCount, &result.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to move guild warehouse item: %w", err)
	}
	return result, nil
}

func (r *GuildRepository) DropGuildWarehouseAtomic(ctx context.Context, cmd appguild.WarehouseDropCommand) (*appguild.WarehouseDropResult, error) {
	result := &appguild.WarehouseDropResult{}
	err := r.db.pool.QueryRow(ctx, "select * from player.drop_guild_warehouse_atomic($1,$2,$3)", cmd.GuildID, cmd.ActorID, cmd.SlotID).
		Scan(&result.TemplateID, &result.StackCount)
	if err != nil {
		return nil, fmt.Errorf("failed to drop guild warehouse item: %w", err)
	}
	return result, nil
}

var _ appguild.WarehouseCommandRepository = (*GuildRepository)(nil)
