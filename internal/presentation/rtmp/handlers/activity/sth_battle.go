package activity

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/infrastructure/rtmp"
)

// Battle start and completion are delegated to the existing combat handler.
func (h *Handler) SetSTHBattleStarter(start func(*rtmp.RPCContext, int) (string, error)) {
	h.sthBattleStarter = start
}

func sthNPCForPosition(pos int) (int, error) {
	switch pos {
	case 5, 28, 59, 83:
		return 2348, nil
	case 12, 36, 66, 91:
		return 2349, nil
	case 20, 44, 70, 99:
		return 2350, nil
	default:
		return 0, fmt.Errorf("STH: position %d has no NPC group", pos)
	}
}

func (h *Handler) sthStartEncounter(ctx *rtmp.RPCContext, cid int64, pos int) {
	if h.sthBattleStarter == nil {
		h.sthBattleStartFailed(ctx, cid, pos, "Bí Cảnh: chưa kết nối hệ thống chiến đấu")
		return
	}
	npcID, err := sthNPCForPosition(pos)
	if err != nil {
		h.sthBattleStartFailed(ctx, cid, pos, err.Error())
		return
	}
	battleID, err := h.sthBattleStarter(ctx, npcID)
	if err != nil {
		h.sthBattleStartFailed(ctx, cid, pos, "Không thể khởi tạo trận Bí Cảnh: "+err.Error())
		return
	}
	tag, err := h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='active',battle_id=$4,updated_at=now()
         WHERE character_id=$1 AND activity_day=(SELECT refresh_date FROM player.character_sth_progress WHERE character_id=$1)
           AND position=$2 AND status='starting' AND npc_id=$3`, cid, pos, npcID, battleID)
	if err != nil || tag.RowsAffected() != 1 {
		h.sendActivityCallback(ctx, "onRedMsg", "Trận Bí Cảnh đã mở nhưng không lưu được trạng thái; không được tiếp tục di chuyển")
	}
}

func (h *Handler) sthBattleStartFailed(ctx *rtmp.RPCContext, cid int64, pos int, msg string) {
	_, _ = h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='failed',updated_at=now() WHERE character_id=$1 AND position=$2 AND status='starting'`, cid, pos)
	h.sendActivityCallback(ctx, "onRedMsg", msg)
	h.sendActivityCallback(ctx, "onSetSecTreaHuntCanPlay", true, 0)
}

// Recover committed but not-yet-delivered rewards on map reopen.
func (h *Handler) sthRecoverBattleRewards(ctx *rtmp.RPCContext, cid int64) {
	rows, err := h.findbackPool.Query(ctx.Context, `SELECT activity_day,position FROM player.sth_battle_runs WHERE character_id=$1 AND status='pending'`, cid)
	if err != nil {
		return
	}
	type pending struct {
		day time.Time
		pos int
	}
	var claims []pending
	for rows.Next() {
		var v pending
		if rows.Scan(&v.day, &v.pos) == nil {
			claims = append(claims, v)
		}
	}
	rows.Close()
	for _, v := range claims {
		h.sthDeliverBattleReward(ctx, cid, v.day, v.pos)
	}
}

// Invoked only by combat.EndBattle after the authoritative BattleResult is available.
func (h *Handler) CompleteSTHBattle(ctx *rtmp.RPCContext, cid int64, battleID string, won bool) error {
	if h.findbackPool == nil || battleID == "" {
		return nil
	}
	tx, err := h.findbackPool.Begin(ctx.Context)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx.Context)
	var pos int
	var activityDay time.Time
	err = tx.QueryRow(ctx.Context, `SELECT position,activity_day FROM player.sth_battle_runs
        WHERE character_id=$1 AND battle_id=$2 AND status='active' FOR UPDATE`, cid, battleID).Scan(&pos, &activityDay)
	if err != nil {
		// Normal battles do not have a matching STH run.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if !won {
		_, err = tx.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='lost',updated_at=now()
            WHERE character_id=$1 AND activity_day=$2 AND position=$3`, cid, activityDay, pos)
	} else {
		var itemID, qty int
		err = tx.QueryRow(ctx.Context, `WITH p AS (
                SELECT item_id,quantity,sum(weight) OVER (ORDER BY item_id) cumulative
                FROM player.sth_battle_reward_pool WHERE enabled
            ), total AS (SELECT sum(weight) weight FROM player.sth_battle_reward_pool WHERE enabled)
            SELECT p.item_id,p.quantity FROM p,total
            WHERE p.cumulative >= ($1::bigint % total.weight)+1 ORDER BY p.cumulative LIMIT 1`, randomSTHInt()).Scan(&itemID, &qty)
		if err != nil {
			return fmt.Errorf("STH combat rewards not configured: %w", err)
		}
		_, err = tx.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='pending',item_id=$4,quantity=$5,updated_at=now()
            WHERE character_id=$1 AND activity_day=$2 AND position=$3`, cid, activityDay, pos, itemID, qty)
		if err == nil {
			_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress
                SET visited=array_append(visited,$2),updated_at=now()
                WHERE character_id=$1 AND NOT ($2=ANY(visited))`, cid, pos)
		}
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return err
	}
	if won {
		h.sthDeliverBattleReward(ctx, cid, activityDay, pos)
		h.sendActivityCallback(ctx, "onRedMsg", "Chiến thắng quái Bí Cảnh")
	} else {
		h.sendActivityCallback(ctx, "onRedMsg", "Thất bại trong trận Bí Cảnh")
	}
	h.sendActivityCallback(ctx, "onSetSecTreaHuntCanPlay", true, 0)
	return nil
}

func (h *Handler) sthDeliverBattleReward(ctx *rtmp.RPCContext, cid int64, day time.Time, pos int) {
	if h.itemService == nil {
		return
	}
	tag, err := h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='processing',updated_at=now()
        WHERE character_id=$1 AND activity_day=$2 AND position=$3 AND status='pending'`, cid, day, pos)
	if err != nil || tag.RowsAffected() != 1 {
		return
	}
	var itemID, qty int
	err = h.findbackPool.QueryRow(ctx.Context, `SELECT item_id,quantity FROM player.sth_battle_runs
        WHERE character_id=$1 AND activity_day=$2 AND position=$3`, cid, day, pos).Scan(&itemID, &qty)
	if err != nil {
		return
	}
	award, err := h.itemService.AddItemWithBind(ctx.Context, cid, itemID, h.resolveSendCombineItemType(itemID, 0), qty, true)
	if err != nil {
		_, _ = h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='pending' WHERE character_id=$1 AND activity_day=$2 AND position=$3 AND status='processing'`, cid, day, pos)
		h.sendActivityCallback(ctx, "onRedMsg", "Quà Bí Cảnh đang chờ nhận (kiểm tra túi)")
		return
	}
	_, err = h.findbackPool.Exec(ctx.Context, `UPDATE player.sth_battle_runs SET status='delivered',updated_at=now() WHERE character_id=$1 AND activity_day=$2 AND position=$3 AND status='processing'`, cid, day, pos)
	if err != nil {
		return
	}
	h.sendActivityItemRewardCallbacks(ctx, award)
}
