package activity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"mcgame-server/internal/infrastructure/rtmp"
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
	if moving || xy <= 0 || pos+steps > 103 || (block != nil && block.After(time.Now())) {
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
func (h *Handler) STHStopWithEvent(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if !sthPreviewEnabled() {
		return false, nil
	}
	_, cid, err := h.characterID(ctx, "afterSTHPlayerStop")
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
	err = tx.QueryRow(ctx.Context, `SELECT position,moving FROM player.character_sth_progress WHERE character_id=$1 FOR UPDATE`, cid).Scan(&pos, &moving)
	if err != nil {
		return nil, err
	}
	if !moving {
		return true, nil
	} // Duplicate stop: no double award.
	var eventType int
	err = tx.QueryRow(ctx.Context, `SELECT t.type::integer FROM data.data_tbl_mevent_map m JOIN data.data_tbl_mevent_type t ON t.id=m.mevent_id::integer WHERE m.id=$1`, pos).Scan(&eventType)
	if err != nil {
		return nil, err
	}
	var target *int
	var blockedSeconds int
	_ = tx.QueryRow(ctx.Context, `SELECT blocked_seconds,target_position FROM player.sth_event_settings WHERE event_type=$1`, eventType).Scan(&blockedSeconds, &target)
	if eventType == 9 && pos == 57 {
		dest := 49
		target = &dest
	}
	dest := pos
	if target != nil {
		dest = *target
	}
	_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress SET moving=false,position=$2,blocked_until=CASE WHEN $3::integer>0 THEN now()+($3::integer * interval '1 second') ELSE NULL END,updated_at=now() WHERE character_id=$1`, cid, dest, blockedSeconds)
	if err != nil {
		return nil, err
	}
	first := false
	if eventType >= 4 && eventType <= 6 {
		tag, execErr := tx.Exec(ctx.Context, `INSERT INTO player.sth_event_claim(character_id,position,event_type) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, cid, pos, eventType)
		if execErr != nil {
			return nil, execErr
		}
		first = tag.RowsAffected() == 1
		if first {
			var count int64
			err = tx.QueryRow(ctx.Context, `SELECT count(*) FROM player.sth_reward_pool WHERE tier=$1 AND enabled`, eventType).Scan(&count)
			if err != nil {
				return nil, err
			}
			if count == 0 {
				return nil, fmt.Errorf("STH loot tier %d has no enabled rewards", eventType)
			}
			var itemID, qty int
			err = tx.QueryRow(ctx.Context, `WITH p AS (SELECT item_id,quantity,weight,sum(weight) OVER (ORDER BY item_id) AS cumulative FROM player.sth_reward_pool WHERE tier=$1 AND enabled), t AS (SELECT sum(weight) total FROM player.sth_reward_pool WHERE tier=$1 AND enabled) SELECT p.item_id,p.quantity FROM p,t WHERE p.cumulative >= $2::bigint % t.total+1 ORDER BY p.cumulative LIMIT 1`, eventType, randomSTHInt()).Scan(&itemID, &qty)
			if err != nil {
				return nil, err
			}
			_, err = tx.Exec(ctx.Context, `UPDATE player.sth_event_claim SET item_id=$3,quantity=$4 WHERE character_id=$1 AND position=$2`, cid, pos, itemID, qty)
			if err != nil {
				return nil, err
			}
		}
	}
	if eventType != 1 && eventType != 2 && eventType != 3 && eventType != 9 && eventType != 13 {
		if eventType >= 14 && eventType <= 17 {
			_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress SET visited=array_append(visited,$2) WHERE character_id=$1 AND NOT ($2=ANY(visited))`, cid, pos)
			if err != nil {
				return nil, err
			}
		}
	}
	if eventType >= 4 && eventType <= 6 && first {
		_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress SET visited=array_append(visited,$2) WHERE character_id=$1 AND NOT ($2=ANY(visited))`, cid, pos)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}
	if eventType >= 4 && eventType <= 6 {
		h.sthDeliverPending(ctx, cid, pos)
	}
	if eventType == 9 || eventType == 13 {
		h.sendActivityCallback(ctx, "onSetSecTreaHuntPlayStop", eventType, dest)
	}
	if eventType >= 14 && eventType <= 17 {
		h.sendActivityCallback(ctx, "onRedMsg", fmt.Sprintf("Bị mắc bẫy tại ô %d. Hãy mở lại Bí Cảnh sau khi hết thời gian khóa.", pos))
	}
	if eventType == 8 {
		h.sendActivityCallback(ctx, "onRedMsg", "Ô Đại Thánh: chiến đấu Bí Cảnh chưa cấu hình, chưa phát thưởng")
	}
	if eventType == 11 {
		h.sendActivityCallback(ctx, "onRedMsg", "Minigame Bí Cảnh chưa mở")
	}
	h.sendActivityCallback(ctx, "onSetSecTreaHuntCanPlay", blockedSeconds == 0, 0)
	return true, nil
}

func randomSTHInt() int64 {
	n, e := rand.Int(rand.Reader, big.NewInt(9223372036854775807))
	if e != nil {
		return time.Now().UnixNano()
	}
	return n.Int64()
}

func (h *Handler) sthDeliverPending(ctx *rtmp.RPCContext, cid int64, pos int) {
	if h.itemService == nil {
		h.sendActivityCallback(ctx, "onRedMsg", "Phần thưởng đang chờ: dịch vụ túi không khả dụng")
		return
	}
	// claim ownership prevents duplicate concurrent delivery; pending recovery on next login
	tag, err := h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_event_claim SET state='processing' WHERE character_id=$1 AND position=$2 AND state='pending'`, cid, pos)
	if err != nil || tag.RowsAffected() == 0 {
		return
	}
	var itemID, qty, typ int
	err = h.findbackPool.QueryRow(ctx.Context, `SELECT item_id,quantity,event_type FROM player.sth_event_claim WHERE character_id=$1 AND position=$2`, cid, pos).Scan(&itemID, &qty, &typ)
	if err != nil {
		return
	}
	award, err := h.itemService.AddItemWithBind(ctx.Context, cid, itemID, h.resolveSendCombineItemType(itemID, 0), qty, true)
	if err != nil {
		_, _ = h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_event_claim SET state='pending' WHERE character_id=$1 AND position=$2 AND state='processing'`, cid, pos)
		h.sendActivityCallback(ctx, "onRedMsg", "Túi đầy hoặc lỗi cấp thưởng, phần thưởng đang chờ nhận")
		return
	}
	_, err = h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_event_claim SET state='delivered',delivered_at=now() WHERE character_id=$1 AND position=$2`, cid, pos)
	if err != nil {
		return
	}
	h.sendActivityItemRewardCallbacks(ctx, award)
	h.sendActivityCallback(ctx, "onSetSecTreaHuntPlayStop", typ, itemID)
}
