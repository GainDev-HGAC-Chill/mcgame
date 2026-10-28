// Targeted guild metadata writes preserve concurrent economy changes.
package postgres

import (
	"context"
	"fmt"
)

func (r *GuildRepository) SetGuildName(ctx context.Context, guildID, actorID int64, name string) error {
	var updated bool
	if err := r.db.pool.QueryRow(ctx, "select player.set_guild_name($1,$2,$3)", guildID, actorID, name).Scan(&updated); err != nil {
		return fmt.Errorf("failed to set guild name: %w", err)
	}
	if !updated {
		return fmt.Errorf("guild name update rejected")
	}
	return nil
}

func (r *GuildRepository) SetGuildNotice(ctx context.Context, guildID, actorID int64, announcement, description string) error {
	var updated bool
	if err := r.db.pool.QueryRow(ctx, "select player.set_guild_notice($1,$2,$3,$4)", guildID, actorID, announcement, description).Scan(&updated); err != nil {
		return fmt.Errorf("failed to set guild notice: %w", err)
	}
	if !updated {
		return fmt.Errorf("guild notice update rejected")
	}
	return nil
}
