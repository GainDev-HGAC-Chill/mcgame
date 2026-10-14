package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *StatFeatureRepository) CommitBloodBagItemRefill(
	ctx context.Context,
	charID, itemID int64,
	templateID, bagType, count int,
	unitAmount int64,
) (map[string]interface{}, int, error) {
	if charID <= 0 || itemID <= 0 || count <= 0 ||
		unitAmount <= 0 || bagType < 1 || bagType > 4 {
		return nil, 0, fmt.Errorf("blood bag: invalid refill")
	}

	const capacity int64 = 10_000_000
	if int64(count) > capacity/unitAmount {
		return nil, 0, fmt.Errorf("blood bag: capacity exceeded")
	}

	keys := []string{
		"",
		"characterHp",
		"characterMp",
		"petHp",
		"petMp",
	}
	key := keys[bagType]
	amount := int64(count) * unitAmount

	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Tạo trạng thái rỗng nếu nhân vật chưa có blood_bag.
	_, err = tx.Exec(ctx, `
		INSERT INTO player.character_stat_features
			(character_id, feature_key, state)
		VALUES ($1, 'blood_bag', '{}'::jsonb)
		ON CONFLICT (character_id, feature_key) DO NOTHING
	`, charID)
	if err != nil {
		return nil, 0, err
	}

	// Khóa trạng thái để không có hai lần nạp cùng
	// sử dụng một số dư cũ.
	var current int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE((state ->> $2)::bigint, 0)
		FROM player.character_stat_features
		WHERE character_id = $1
		  AND feature_key = 'blood_bag'
		FOR UPDATE
	`, charID, key).Scan(&current)
	if err != nil {
		return nil, 0, err
	}

	// Khóa đúng vật phẩm thuộc về nhân vật.
	var actualTemplateID, stackCount int
	err = tx.QueryRow(ctx, `
		SELECT template_id, stack_count
		FROM player.character_items
		WHERE id = $1 AND character_id = $2
		FOR UPDATE
	`, itemID, charID).Scan(&actualTemplateID, &stackCount)

	if err == pgx.ErrNoRows {
		return nil, 0, fmt.Errorf("blood bag: item not found")
	}
	if err != nil {
		return nil, 0, err
	}
	if actualTemplateID != templateID || stackCount < count {
		return nil, 0, fmt.Errorf("blood bag: invalid item or quantity")
	}

	if current < 0 || current > capacity ||
		amount > capacity-current {
		return nil, 0, fmt.Errorf("blood bag: capacity exceeded")
	}

	// Cộng dự trữ.
	var rawState []byte
	err = tx.QueryRow(ctx, `
		UPDATE player.character_stat_features
		SET state = jsonb_set(
				state,
				ARRAY[$2::text],
				to_jsonb($3::bigint),
				true
			),
			updated_at = now()
		WHERE character_id = $1
		  AND feature_key = 'blood_bag'
		RETURNING state
	`, charID, key, current+amount).Scan(&rawState)
	if err != nil {
		return nil, 0, err
	}

	// Trừ vật phẩm trong cùng transaction.
	remaining := stackCount - count

	if remaining == 0 {
		_, err = tx.Exec(ctx, `
			DELETE FROM player.character_items
			WHERE id = $1 AND character_id = $2
		`, itemID, charID)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE player.character_items
			SET stack_count = $3,
				updated_at = now()
			WHERE id = $1 AND character_id = $2
		`, itemID, charID, remaining)
	}
	if err != nil {
		return nil, 0, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(rawState, &state); err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}

	return state, remaining, nil
}
