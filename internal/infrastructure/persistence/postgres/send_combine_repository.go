// PostgreSQL Send Combine commands return only committed purchase state.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	appactivity "mcgame-server/internal/application/activity"
	domainactivity "mcgame-server/internal/domain/activity"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type SendCombineRepository struct {
	db *Database
}

func NewSendCombineRepository(db *Database) *SendCombineRepository {
	return &SendCombineRepository{db: db}
}

func (r *SendCombineRepository) PurchaseSendCombine(ctx context.Context, command domainactivity.SendCombinePurchaseCommand) (*domainactivity.SendCombinePurchaseResult, error) {
	awardsJSON, err := json.Marshal(command.Awards)
	if err != nil {
		return nil, fmt.Errorf("encode Send Combine awards: %w", err)
	}
	result := &domainactivity.SendCombinePurchaseResult{}
	var itemsJSON []byte
	err = r.db.pool.QueryRow(ctx,
		"select * from player.purchase_send_combine_atomic($1, $2, $3, $4, $5, $6::jsonb)",
		command.CharacterID, command.ActivityID, command.EventEndMs, command.Limit, command.Cost, awardsJSON,
	).Scan(&result.BuyCount, &result.Gold, &itemsJSON)
	if err != nil {
		return nil, mapSendCombineCommandError(err)
	}
	if err := json.Unmarshal(itemsJSON, &result.Items); err != nil {
		return nil, fmt.Errorf("decode committed Send Combine items: %w", err)
	}
	for _, committed := range result.Items {
		if committed == nil || committed.ID <= 0 || committed.CharacterID != command.CharacterID {
			return nil, pkgerrors.ErrSystemError
		}
		if committed.Properties == nil {
			committed.Properties = make(map[string]interface{})
		}
	}
	return result, nil
}

func (r *SendCombineRepository) ListSendCombineBuyCounts(ctx context.Context, characterID int64, eventEndMs int64) (map[string]int, error) {
	rows, err := r.db.pool.Query(ctx, `
		select activity_id, buy_count
		from player.character_send_combine_purchases
		where character_id = $1 and event_end_ms = $2`, characterID, eventEndMs)
	if err != nil {
		return nil, fmt.Errorf("list Send Combine purchase counts: %w", err)
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var activityID string
		var buyCount int
		if err := rows.Scan(&activityID, &buyCount); err != nil {
			return nil, fmt.Errorf("scan Send Combine purchase count: %w", err)
		}
		result[activityID] = buyCount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read Send Combine purchase counts: %w", err)
	}
	return result, nil
}

func mapSendCombineCommandError(err error) error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Message {
		case "SEND_COMBINE_EXPIRED":
			return appactivity.ErrSendCombineExpired
		case "SEND_COMBINE_LIMIT_REACHED":
			return appactivity.ErrSendCombineLimitReached
		case "INSUFFICIENT_FUNDS":
			return pkgerrors.ErrInsufficientFunds
		case "INVENTORY_FULL":
			return pkgerrors.ErrInventoryFull
		case "CHARACTER_NOT_FOUND":
			return pkgerrors.ErrCharacterNotFound
		case "INVALID_SEND_COMBINE_COMMAND", "INVALID_SEND_COMBINE_AWARD":
			return pkgerrors.ErrInvalidInput
		}
	}
	return fmt.Errorf("purchase Send Combine atomically: %w", err)
}

var _ domainactivity.SendCombineRepository = (*SendCombineRepository)(nil)
