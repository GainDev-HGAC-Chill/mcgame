package activity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"
	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/infrastructure/rtmp"
	appitem "mcgame-server/internal/application/item"
)

func sthIntArg(value interface{}) (int, bool) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v || v > 2147483647 || v < -2147483648 {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case int64:
		if v > 2147483647 || v < -2147483648 {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}

func (h *Handler) STHBuyDice(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if !sthPreviewEnabled() {
		return false, nil
	}
	if len(args) != 2 {
		return false, errors.New("STH buy: expected dice type and quantity")
	}
	kind, ok := sthIntArg(args[0])
	if !ok || (kind != 1 && kind != 2) {
		return false, errors.New("STH buy: invalid dice type")
	}
	quantity, ok := sthIntArg(args[1])
	if !ok || quantity < 1 || quantity > 50 {
		return false, errors.New("STH buy: quantity must be 1..50")
	}
	_, cid, err := h.characterID(ctx, "buySecTreaHuntSZ")
	if err != nil {
		return nil, err
	}
	if err = h.sthPrepare(ctx, cid); err != nil {
		return nil, err
	}
	tx, err := h.findbackPool.Begin(ctx.Context)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx.Context)
	var bought, pt, xy int
	err = tx.QueryRow(ctx.Context, `SELECT bought_dice,normal_dice,lucky_dice FROM player.character_sth_progress WHERE character_id=$1 FOR UPDATE`, cid).Scan(&bought, &pt, &xy)
	if err != nil {
		return nil, err
	}
	if bought+quantity > 50 {
		h.sendActivityCallback(ctx, "onRedMsg", "Đã đạt giới hạn mua 50 xúc xắc")
		return false, nil
	}
	price := int64(20)
	if kind == 2 {
		price = 50
	}
	cost := price * int64(quantity)
	var gold int64
	err = tx.QueryRow(ctx.Context, `UPDATE player.character_wallet SET gold=gold-$2 WHERE character_id=$1 AND gold >= $2 RETURNING gold`, cid, cost).Scan(&gold)
	if err != nil {
		h.sendActivityCallback(ctx, "onRedMsg", "Không đủ vàng để mua xúc xắc")
		return false, nil
	}
	if kind == 1 {
		pt += quantity
	} else {
		xy += quantity
	}
	_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress SET normal_dice=$2,lucky_dice=$3,bought_dice=bought_dice+$4,updated_at=now() WHERE character_id=$1`, cid, pt, xy, quantity)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}
	h.sendActivityCallback(ctx, "onSetSecTreaHuntSZnum", map[string]interface{}{"pt": pt, "xy": xy, "all": bought + quantity})
	h.sendActivityCallback(ctx, "onAddMoney", float64(cid), "gold", float64(-cost), float64(gold))
	h.sendActivityCallback(ctx, "onUPP", map[string]interface{}{"gold": gold})
	return true, nil
}

func (h *Handler) STHGoXY(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if !sthPreviewEnabled() {
		return map[string]interface{}{"num": 0}, nil
	}
	if len(args) != 1 {
		return map[string]interface{}{"num": 0}, nil
	}
	steps, ok := sthIntArg(args[0])
	if !ok || steps < 1 || steps > 6 {
		return map[string]interface{}{"num": 0}, nil
	}
	_, cid, err := h.characterID(ctx, "SecretTreasureHuntGoXY")
	if err != nil {
		return nil, err
	}
	if err = h.sthPrepare(ctx, cid); err != nil {
		return nil, err
	}
	tx, err := h.findbackPool.Begin(ctx.Context)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx.Context)
	var pos, xy int
	var moving bool
	var block *time.Time
	err = tx.QueryRow(ctx.Context, `SELECT position,lucky_dice,moving,blocked_until FROM player.character_sth_progress WHERE character_id=$1 FOR UPDATE`, cid).Scan(&pos, &xy, &moving, &block)
	if err != nil {
		return nil, err
	}
	var pendingSTHFight bool
	err = tx.QueryRow(ctx.Context, `SELECT EXISTS(SELECT 1 FROM player.sth_battle_runs r
    WHERE r.character_id=$1 AND r.activity_day=(SELECT refresh_date FROM player.character_sth_progress WHERE character_id=$1)
      AND r.status IN ('starting','active'))`, cid).Scan(&pendingSTHFight)
	if err != nil {
		return nil, err
	}
	if moving || pendingSTHFight || xy <= 0 || pos+steps > 103 || (block != nil && block.After(time.Now())) {
		return map[string]interface{}{"num": 0}, nil
	}
	_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress SET position=$2,lucky_dice=lucky_dice-1,moving=true,updated_at=now() WHERE character_id=$1`, cid, pos+steps)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}
	return map[string]interface{}{"num": steps}, nil
}

// Event stop is atomic with position/state changes. Item delivery is intentionally
// separate via the existing inventory service, with durable pending claims.

func (h *Handler) STHStopWithEvent(
	ctx *rtmp.RPCContext,
	args []interface{},
) (interface{}, error) {
	if !sthPreviewEnabled() {
		return false, nil
	}

	_, cid, err := h.characterID(
		ctx,
		"afterSTHPlayerStop",
	)
	if err != nil {
		return nil, err
	}

	if h.findbackPool == nil {
		return nil, errors.New("STH database unavailable")
	}

	tx, err := h.findbackPool.Begin(ctx.Context)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx.Context)

	var pos int
	var moving bool

	err = tx.QueryRow(ctx.Context, `
		SELECT position, moving
		FROM player.character_sth_progress
		WHERE character_id = $1
		FOR UPDATE
	`, cid).Scan(&pos, &moving)

	if err != nil {
		return nil, err
	}

	// Ngăn callback lặp phát thưởng hai lần.
	if !moving {
		return true, nil
	}

	// Xác định sự kiện tại vị trí hiện tại.
	var eventType int

	err = tx.QueryRow(ctx.Context, `
		SELECT t.type::integer
		FROM data.data_tbl_mevent_map m
		JOIN data.data_tbl_mevent_type t
		  ON t.id = m.mevent_id::integer
		WHERE m.id = $1
	`, pos).Scan(&eventType)

	if err != nil {
		return nil, err
	}

	// Lấy cấu hình dịch chuyển hoặc khóa di chuyển.
	var target *int
	var blockedSeconds int

	_ = tx.QueryRow(ctx.Context, `
		SELECT blocked_seconds, target_position
		FROM player.sth_event_settings
		WHERE event_type = $1
	`, eventType).Scan(
		&blockedSeconds,
		&target,
	)

	if eventType == 9 && pos == 57 {
		destination := 49
		target = &destination
	}

	dest := pos
	if target != nil {
		dest = *target
	}

	_, err = tx.Exec(ctx.Context, `
		UPDATE player.character_sth_progress
		SET moving = false,
		    position = $2,
		    blocked_until = CASE
		        WHEN $3::integer > 0
		        THEN now() + ($3::integer * interval '1 second')
		        ELSE NULL
		    END,
		    updated_at = now()
		WHERE character_id = $1
	`, cid, dest, blockedSeconds)

	if err != nil {
		return nil, err
	}

	// Xử lý các ô nhận thưởng (eventType 4, 5, 6).
	first := false

	if eventType >= 4 && eventType <= 6 {
		tag, insertErr := tx.Exec(ctx.Context, `
			INSERT INTO player.sth_event_claim (
				character_id,
				position,
				event_type
			)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, cid, pos, eventType)

		if insertErr != nil {
			return nil, insertErr
		}

		first = tag.RowsAffected() == 1

		if first {
			var tableType, itemID, qty int

			// Quay thưởng theo trọng số.
			// Mỗi phần thưởng có TableType rõ ràng.
			err = tx.QueryRow(ctx.Context, `
				WITH rewards AS (
					SELECT
						table_type,
						item_id,
						quantity,
						SUM(weight) OVER (
							ORDER BY table_type, item_id
						) AS cumulative
					FROM player.sth_reward_pool
					WHERE tier = $1
					  AND enabled = true
				),
				total AS (
					SELECT SUM(weight) AS total_weight
					FROM player.sth_reward_pool
					WHERE tier = $1
					  AND enabled = true
				)
				SELECT
					r.table_type,
					r.item_id,
					r.quantity
				FROM rewards r
				CROSS JOIN total t
				WHERE r.cumulative >=
				      ($2::bigint % t.total_weight) + 1
				ORDER BY r.cumulative
				LIMIT 1
			`,
				eventType,
				randomSTHInt(),
			).Scan(
				&tableType,
				&itemID,
				&qty,
			)

			if err != nil {
				return nil, fmt.Errorf(
					"STH loot tier %d has no valid rewards: %w",
					eventType,
					err,
				)
			}

			// Ghi phần thưởng đã chọn, giữ nguyên
			// TableType để sau này nhận lại đúng vật phẩm.
			tag, err = tx.Exec(ctx.Context, `
				UPDATE player.sth_event_claim
				SET table_type = $3,
				    item_id = $4,
				    quantity = $5
				WHERE character_id = $1
				  AND position = $2
				  AND state = 'pending'
			`,
				cid,
				pos,
				tableType,
				itemID,
				qty,
			)

			if err != nil {
				return nil, err
			}
			if tag.RowsAffected() != 1 {
				return nil, fmt.Errorf(
					"STH event reward claim changed unexpectedly",
				)
			}
		}
	}

	// Xử lý các ô bẫy.
	if eventType >= 14 && eventType <= 17 {
		_, err = tx.Exec(ctx.Context, `
			UPDATE player.character_sth_progress
			SET visited = array_append(visited, $2)
			WHERE character_id = $1
			  AND NOT ($2 = ANY(visited))
		`, cid, pos)

		if err != nil {
			return nil, err
		}
	}

	// Chỉ đánh dấu hoàn thành nếu lần đầu nhận thưởng.
	if eventType >= 4 && eventType <= 6 && first {
		_, err = tx.Exec(ctx.Context, `
			UPDATE player.character_sth_progress
			SET visited = array_append(visited, $2)
			WHERE character_id = $1
			  AND NOT ($2 = ANY(visited))
		`, cid, pos)

		if err != nil {
			return nil, err
		}
	}

	// Xử lý sự kiện chiến đấu.
	encounterStarts := false

	if eventType == 8 {
		npcID, npcErr := sthNPCForPosition(pos)
		if npcErr != nil {
			return nil, npcErr
		}

		tag, insertErr := tx.Exec(ctx.Context, `
			INSERT INTO player.sth_battle_runs (
				character_id,
				activity_day,
				position,
				npc_id,
				status
			)
			SELECT
				$1,
				refresh_date,
				$2,
				$3,
				'starting'
			FROM player.character_sth_progress
			WHERE character_id = $1
			ON CONFLICT (
				character_id,
				activity_day,
				position
			) DO NOTHING
		`, cid, pos, npcID)

		if insertErr != nil {
			return nil, insertErr
		}

		encounterStarts = tag.RowsAffected() == 1
	}

	// Hoàn tất transaction trước khi phát thưởng.
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}

	// Phát thưởng sự kiện.
	if eventType >= 4 && eventType <= 6 {
		h.sthDeliverPending(ctx, cid, pos)
	}

	// Xử lý dịch chuyển.
	if eventType == 9 || eventType == 13 {
		h.sendActivityCallback(
			ctx,
			"onSetSecTreaHuntPlayStop",
			eventType,
			dest,
		)
	}

	// Thông báo bẫy.
	if eventType >= 14 && eventType <= 17 {
		h.sendActivityCallback(
			ctx,
			"onRedMsg",
			fmt.Sprintf(
				"Bị mắc bẫy tại ô %d. Hãy mở lại Bí Cảnh sau khi hết thời gian khóa.",
				pos,
			),
		)
	}

	// Khởi tạo trận đấu.
	if eventType == 8 && encounterStarts {
		h.sthStartEncounter(ctx, cid, pos)
	}

	// Minigame chưa được triển khai.
	if eventType == 11 {
		h.sendActivityCallback(
			ctx,
			"onRedMsg",
			"Minigame Bí Cảnh chưa mở",
		)
	}

	h.sendActivityCallback(
		ctx,
		"onSetSecTreaHuntCanPlay",
		blockedSeconds == 0 &&
			(eventType != 8 || !encounterStarts),
		0,
	)

	return true, nil
}


func randomSTHInt() int64 {
	n, e := rand.Int(rand.Reader, big.NewInt(9223372036854775807))
	if e != nil {
		return time.Now().UnixNano()
	}
	return n.Int64()
}


func (h *Handler) sthDeliverPending(
	ctx *rtmp.RPCContext,
	cid int64,
	pos int,
) {
	if h.itemService == nil || h.findbackPool == nil {
		h.sendActivityCallback(
			ctx,
			"onRedMsg",
			"Phần thưởng đang chờ: dịch vụ túi không khả dụng",
		)
		return
	}

	var tableType, itemID, qty, typ int

	// Nhận quyền xử lý và lấy thông tin thưởng.
	// Chỉ xử lý bản ghi đang ở trạng thái pending.
	err := h.findbackPool.QueryRow(ctx.Context, `
		UPDATE player.sth_event_claim
		SET state = 'processing'
		WHERE character_id = $1
		  AND position = $2
		  AND state = 'pending'
		  AND table_type IN (19, 29)
		  AND item_id > 0
		  AND quantity > 0
		RETURNING
			table_type,
			item_id,
			quantity,
			event_type
	`, cid, pos).Scan(
		&tableType,
		&itemID,
		&qty,
		&typ,
	)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			h.sendActivityCallback(
				ctx,
				"onRedMsg",
				"Lỗi đọc phần thưởng sự kiện Bí Cảnh",
			)
		}
		return
	}

	// Phát vật phẩm theo đúng bảng template.
	award, err := h.itemService.AddTemplateItem(
		ctx.Context,
		cid,
		appitem.TemplateRef{
			TableType:  tableType,
			TemplateID: itemID,
		},
		qty,
		true,
		0,
	)

	if err != nil {
		// Nếu phát vật phẩm thất bại, cho phép thử lại.
		_, resetErr := h.findbackPool.Exec(ctx.Context, `
			UPDATE player.sth_event_claim
			SET state = 'pending'
			WHERE character_id = $1
			  AND position = $2
			  AND state = 'processing'
		`, cid, pos)

		if resetErr != nil {
			h.sendActivityCallback(
				ctx,
				"onRedMsg",
				"Lỗi khôi phục phần thưởng sự kiện Bí Cảnh",
			)
			return
		}

		h.sendActivityCallback(
			ctx,
			"onRedMsg",
			"Túi đầy hoặc lỗi cấp thưởng, phần thưởng đang chờ nhận",
		)
		return
	}

	// Xác nhận phần thưởng đã được phát.
	tag, err := h.findbackPool.Exec(ctx.Context, `
		UPDATE player.sth_event_claim
		SET state = 'delivered',
		    delivered_at = now()
		WHERE character_id = $1
		  AND position = $2
		  AND state = 'processing'
	`, cid, pos)

	if err != nil || tag.RowsAffected() != 1 {
		// Tránh đưa về pending vì vật phẩm có thể
		// đã được tạo trong túi.
		h.sendActivityCallback(
			ctx,
			"onRedMsg",
			"Đã phát vật phẩm nhưng chưa xác nhận được trạng thái thưởng. Cần kiểm tra dữ liệu.",
		)
		return
	}

	// Đồng bộ túi đồ và thông báo tới Flash client.
	h.sendActivityItemRewardCallbacks(ctx, award)

	h.sendActivityCallback(
		ctx,
		"onSetSecTreaHuntPlayStop",
		typ,
		itemID,
	)
}

