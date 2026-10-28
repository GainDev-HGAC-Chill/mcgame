// PostgreSQL command for atomic guild skill development.
package postgres

import (
	"context"
	"fmt"

	appguild "mcgame-server/internal/application/guild"
)

func (r *GuildRepository) DevelopGuildSkillAtomic(ctx context.Context, guildID, actorID int64, development appguild.GuildSkillDevelopment) error {
	var applied bool
	err := r.db.pool.QueryRow(ctx, "select player.develop_guild_skill_atomic($1,$2,$3,$4,$5,$6,$7,$8)",
		guildID, actorID, development.SkillID, development.Previous, development.Level, development.ExpCost, development.MoneyCost, development.FamilyIDs,
	).Scan(&applied)
	if err != nil {
		return fmt.Errorf("failed to develop guild skill atomically: %w", err)
	}
	if !applied {
		return appguild.ErrUnsupportedAction
	}
	return nil
}

var _ appguild.GuildSkillCommandRepository = (*GuildRepository)(nil)
