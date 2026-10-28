// Open-sourced by BaoLT

// Achievement fact writer appends idempotent business facts inside caller-owned transactions.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	domainachievement "mcgame-server/internal/domain/achievement"
	pkgerrors "mcgame-server/pkg/errors"
)

func appendAchievementFact(ctx context.Context, queryer creationQueryer, fact domainachievement.Fact) error {
	if fact.FactID == "" || fact.CharacterID <= 0 || fact.Kind == "" || fact.OccurredAt.IsZero() || !json.Valid(fact.Payload) {
		return fmt.Errorf("invalid achievement fact")
	}
	var stored bool
	if err := queryer.QueryRow(
		ctx,
		"select player.append_character_achievement_fact($1, $2, $3, $4::jsonb, $5)",
		fact.FactID,
		fact.CharacterID,
		fact.Kind,
		[]byte(fact.Payload),
		fact.OccurredAt,
	).Scan(&stored); err != nil {
		return pkgerrors.Wrap(err, "failed to append achievement fact")
	}
	if !stored {
		return fmt.Errorf("achievement fact %q conflicts with existing fact", fact.FactID)
	}
	return nil
}
