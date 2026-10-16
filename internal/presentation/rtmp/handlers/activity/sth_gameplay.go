package activity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"mcgame-server/internal/infrastructure/rtmp"
)

const sthDailyDice = 5 // Test setting: 5 free normal dice per day; tune after validating original economy.

func sthPreviewEnabled() bool { return os.Getenv("MCGAME_STH_PREVIEW") == "1" }
func sthLocalDay() time.Time {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("GMT+7", 7*3600)
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}
func (h *Handler) sthPrepare(ctx *rtmp.RPCContext, cid int64) error {
	if h.findbackPool == nil {
		return errors.New("STH requires database pool: SetFindbackPool")
	}
	_, err := h.findbackPool.Exec(ctx.Context, `
 INSERT INTO player.character_sth_progress(character_id,refresh_date,position,normal_dice,lucky_dice,visited,moving)
 VALUES ($1,$2,1,$3,0,'{}'::integer[],false)
 ON CONFLICT(character_id) DO UPDATE SET
  normal_dice=CASE WHEN character_sth_progress.refresh_date < EXCLUDED.refresh_date THEN $3 ELSE character_sth_progress.normal_dice END,
  refresh_date=GREATEST(character_sth_progress.refresh_date,EXCLUDED.refresh_date)
 `, cid, sthLocalDay(), sthDailyDice)
	return err
}
func (h *Handler) STHGetData(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if !sthPreviewEnabled() {
		return false, nil
	}
	_, cid, err := h.characterID(ctx, "getSecretTreasureHuntData")
	if err != nil {
		return nil, err
	}
	if err = h.sthPrepare(ctx, cid); err != nil {
		return nil, fmt.Errorf("prepare sth: %w", err)
	}
	var position, normal, lucky, bought int
	var visited []int32
	err = h.findbackPool.QueryRow(ctx.Context, `SELECT position,normal_dice,lucky_dice,bought_dice,visited FROM player.character_sth_progress WHERE character_id=$1`, cid).Scan(&position, &normal, &lucky, &bought, &visited)
	if err != nil {
		return nil, err
	}
	haveGo := make([]interface{}, 0, len(visited))
	for _, v := range visited {
		haveGo = append(haveGo, int(v))
	}
	mapState := map[string]interface{}{"nullGridMapData": map[string]interface{}{}, "haveGo": haveGo, "atGridNum": position, "ptSZnum": normal, "xySZnum": lucky, "check": map[string]interface{}{"pt": false, "xy": false}, "allSZBuyNum": bought}
	h.sendActivityCallback(ctx, "onSecretTreasureHuntGetData", mapState, map[string]interface{}{"flag": true, "type": 0}, map[string]interface{}{"play": false})
	return true, nil
}
func (h *Handler) STHGo(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if !sthPreviewEnabled() {
		return map[string]interface{}{"num": 0}, nil
	}
	_, cid, err := h.characterID(ctx, "SecretTreasureHuntGo")
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
	var position, dice int
	var moving bool
	err = tx.QueryRow(ctx.Context, `SELECT position,normal_dice,moving FROM player.character_sth_progress WHERE character_id=$1 FOR UPDATE`, cid).Scan(&position, &dice, &moving)
	if err != nil {
		return nil, err
	}
	if moving || dice <= 0 || position >= 103 {
		return map[string]interface{}{"num": 0}, nil
	}
	maxRoll := 6
	if 103-position < maxRoll {
		maxRoll = 103 - position
	}
	rnd, err := rand.Int(rand.Reader, big.NewInt(int64(maxRoll)))
	if err != nil {
		return nil, err
	}
	rolled := int(rnd.Int64()) + 1
	_, err = tx.Exec(ctx.Context, `UPDATE player.character_sth_progress SET position=$2,normal_dice=normal_dice-1,moving=true,visited=array_append(visited,$2::integer),updated_at=now() WHERE character_id=$1`, cid, position+rolled, position)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}
	return map[string]interface{}{"num": rolled}, nil
}
func (h *Handler) STHStop(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
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
	_, err = h.findbackPool.Exec(ctx.Context, `UPDATE player.character_sth_progress SET moving=false,updated_at=now() WHERE character_id=$1`, cid)
	if err != nil {
		return nil, err
	}
	h.sendActivityCallback(ctx, "onSetSecTreaHuntCanPlay", true, 0)
	return true, nil
}
