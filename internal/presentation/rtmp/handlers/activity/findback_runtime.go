package activity

import (
	"context"
	"errors"
	"fmt"
	"time"

	appactivity "mcgame-server/internal/application/activity"
	domainchar "mcgame-server/internal/domain/character"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/predef"
	"mcgame-server/internal/infrastructure/rtmp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This implementation deliberately refuses to redeem an activity without a
// verified historical activity row. It never fabricates a missed day.
var findbackLimits = map[string]int{"xiuxing": 1, "xueyuan": 2, "chongwu": 1, "panjun": 1, "feimo": 1, "xuanshang": 10}
var findbackKinds = []string{"xiuxing", "xueyuan", "chongwu", "panjun", "feimo"}

func (h *Handler) SetFindbackPool(pool *pgxpool.Pool) { h.findbackPool = pool }

func findbackDay() time.Time {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*3600)
	}
	now := time.Now().In(loc).AddDate(0, 0, -1)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}

func (h *Handler) GetFindBackData(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, charID, err := h.characterID(ctx, "GetFindBackData")
	if err != nil {
		return nil, err
	}
	if h.findbackPool == nil {
		return nil, errors.New("findback repository unavailable")
	}
	day := findbackDay()
	var level int
	if err := h.findbackPool.QueryRow(ctx.Context, `SELECT level FROM player.character_progression WHERE character_id=$1`, charID).Scan(&level); err != nil {
		return nil, err
	}
	num := map[string]interface{}{}
	exp := map[string]interface{}{}
	for _, kind := range findbackKinds {
		limit := findbackLimits[kind]
		// Without evidence of a recorded day: no entitlement.
		completed := limit
		var count, claimed int
		err := h.findbackPool.QueryRow(ctx.Context, `SELECT completed_count, claimed_count FROM player.character_findback_days WHERE character_id=$1 AND activity_day=$2 AND activity_key=$3`, charID, day, kind).Scan(&count, &claimed)
		if err == nil {
			completed = min(limit, max(0, count+claimed))
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		num[kind] = completed
		base := appactivity.FindbackProposedExperience(kind, level)
		// Client expects total EXP for all missing attempts, not EXP per attempt.
		countMissing := limit - completed
		if countMissing <= 0 {
			exp[kind] = int64(0)
		} else {
			exp[kind] = base * int64(countMissing) / int64(limit)
		}
	}
	// Never manufacture bounty claims. The client requires these additional fields.
	num["xuanshang_num"] = 10
	num["xuanshang_refresh"] = 4
	num["xuanshang_color"] = 0
	exp["xuanshang"] = int64(0)
	// Bounty is exposed only with an explicitly verified row and color.
	var bountyDone, bountyClaimed, bountyColor int
	err = h.findbackPool.QueryRow(ctx.Context, `SELECT completed_count,claimed_count,bounty_color FROM player.character_findback_days WHERE character_id=$1 AND activity_day=$2 AND activity_key='xuanshang' AND bounty_color BETWEEN 0 AND 5`, charID, day).Scan(&bountyDone, &bountyClaimed, &bountyColor)
	if err == nil {
		num["xuanshang_num"] = min(10, max(0, bountyDone+bountyClaimed))
		num["xuanshang_color"] = bountyColor
		if bountyDone+bountyClaimed < 10 {
			exp["xuanshang"] = predef.CalcCallboardExp(level, 0)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return map[string]interface{}{"num": num, "exp": exp}, nil
}

func (h *Handler) FindBackXiuXing(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.findbackClaim(ctx, args, "xiuxing", 1)
}
func (h *Handler) FindBackXueYuan(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.findbackClaim(ctx, args, "xueyuan", 2)
}
func (h *Handler) FindBackChongWu(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.findbackPetClaim(ctx, args)
}
func (h *Handler) FindBackPanJun(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.findbackClaim(ctx, args, "panjun", 4)
}
func (h *Handler) FindBackFeiMo(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.findbackClaim(ctx, args, "feimo", 5)
}
func (h *Handler) FindBackXuanShangFree(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.findbackClaim(ctx, args, "xuanshang", 6)
}
func (h *Handler) FindBackXuanShangByBanner(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return nil, nil
} // inventory deduction pending

// Character-only claims: wallet, progression, attributes, and claim journal
// are all updated under one database transaction, with deterministic locks.
func (h *Handler) findbackClaim(ctx *rtmp.RPCContext, args []interface{}, kind string, uiType int) (interface{}, error) {
	_, charID, err := h.characterID(ctx, "FindBackClaim")
	if err != nil {
		return nil, err
	}
	if h.findbackPool == nil {
		return nil, errors.New("findback repository unavailable")
	}
	if len(args) == 0 {
		return nil, nil
	}
	mode, ok := parseIntArg(args[0])
	if !ok || (mode != 1 && mode != 2) {
		return nil, nil
	}
	day := findbackDay()
	tx, err := h.findbackPool.BeginTx(ctx.Context, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx.Context)
	// Canonical lock order: progression -> wallet -> entitlement.
	var level int
	var currentExp int64
	if err = tx.QueryRow(ctx.Context, `SELECT level,experience FROM player.character_progression WHERE character_id=$1 FOR UPDATE`, charID).Scan(&level, &currentExp); err != nil {
		return nil, err
	}
	var money, gold int64
	if err = tx.QueryRow(ctx.Context, `SELECT money,gold FROM player.character_wallet WHERE character_id=$1 FOR UPDATE`, charID).Scan(&money, &gold); err != nil {
		return nil, err
	}
	var completed, claimed int
	err = tx.QueryRow(ctx.Context, `SELECT completed_count,claimed_count FROM player.character_findback_days WHERE character_id=$1 AND activity_day=$2 AND activity_key=$3 FOR UPDATE`, charID, day, kind).Scan(&completed, &claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	limit := findbackLimits[kind]
	missing := limit - completed - claimed
	if missing <= 0 {
		return nil, nil
	}
	if missing > limit {
		return nil, fmt.Errorf("invalid findback count")
	}
	bountyColor := 0
	if kind == "xuanshang" {
		if err = tx.QueryRow(ctx.Context, `SELECT bounty_color FROM player.character_findback_days WHERE character_id=$1 AND activity_day=$2 AND activity_key='xuanshang' AND bounty_color BETWEEN 0 AND 5 FOR UPDATE`, charID, day).Scan(&bountyColor); err != nil {
			return nil, nil
		}
	}
	base := appactivity.FindbackProposedExperience(kind, level) * int64(missing) / int64(limit)
	if kind == "xuanshang" {
		mult := []int64{1, 2, 8, 12, 20, 2}[bountyColor]
		base = predef.CalcCallboardExp(level, 0) * int64(missing) * mult / 10
	}
	awarded := appactivity.FindbackAwardedExperience(base, mode)
	if awarded <= 0 {
		return nil, nil
	}
	cost := appactivity.FindbackCost(kind, mode, missing)
	if kind == "xuanshang" {
		cost = appactivity.FindbackCost(kind, mode, 1)
	} // fixed fee per bounty claim
	if mode == 1 && money < cost {
		return nil, nil
	}
	if mode == 2 && gold < cost {
		return nil, nil
	}
	column := "money"
	if mode == 2 {
		column = "gold"
	}
	if _, err = tx.Exec(ctx.Context, `UPDATE player.character_wallet SET `+column+`=`+column+`-$1 WHERE character_id=$2`, cost, charID); err != nil {
		return nil, err
	}
	// Leverage canonical character leveling thresholds and grant the same attribute bonuses.
	virtual := &domainchar.Character{Level: level, Experience: currentExp}
	// Keep other persisted stats intact: only changes made by GainExperience are applied.
	leveled := virtual.GainExperience(awarded)
	if _, err = tx.Exec(ctx.Context, `UPDATE player.character_progression SET level=$1,experience=$2 WHERE character_id=$3`, virtual.Level, virtual.Experience, charID); err != nil {
		return nil, err
	}
	if leveled {
		delta := virtual.Level - level
		_, err = tx.Exec(ctx.Context, `UPDATE player.character_attributes SET attr_points=attr_points+$1,max_attr_points=max_attr_points+$1,strength=strength+$2,agility=agility+$2,stamina=stamina+$2,intelligence=intelligence+$2,spirit=spirit+$2 WHERE character_id=$3`, delta*domainchar.AttributePointsPerLevel, delta, charID)
		if err != nil {
			return nil, err
		}
	}
	if _, err = tx.Exec(ctx.Context, `UPDATE player.character_findback_days SET claimed_count=claimed_count+$1,updated_at=now() WHERE character_id=$2 AND activity_day=$3 AND activity_key=$4`, missing, charID, day, kind); err != nil {
		return nil, err
	}
	currency := "money"
	if mode == 2 {
		currency = "gold"
	}
	if _, err = tx.Exec(ctx.Context, `INSERT INTO player.character_findback_claims(character_id,activity_day,activity_key,claim_mode,count_claimed,exp_awarded,cost_currency,cost_amount) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, charID, day, kind, mode, missing, awarded, currency, cost); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}
	if ctx.Connection != nil {
		total := money - cost
		if mode == 2 {
			total = gold - cost
		}
		h.sendOfflineCurrencyRefresh(ctx, charID, currency, -cost, total)
		// onUPP: clients update both level and the residual EXP.
		_ = ctx.Connection.SendCallback("onUPP", map[string]interface{}{"expSkill": virtual.Experience})
		if leveled {
			_ = ctx.Connection.SendCallback("onPlayerLevelUp", map[string]interface{}{"id": fmt.Sprint(charID), "level": virtual.Level, "exp": virtual.Experience, "lp": fmt.Sprint((virtual.Level - level) * domainchar.AttributePointsPerLevel), "maxMovePnt": 110, "maxAct": 1160, "maxVigor": 110})
		}
	}
	if leveled && h.eventBus != nil {
		h.eventBus.EmitLevelUp(context.Background(), charID, level, virtual.Level)
	}
	// Client expects completion count rather than number claimed in `num`.
	response := map[string]interface{}{"type": uiType, "num": min(limit, completed+missing), "exp": awarded}
	if kind == "xuanshang" {
		response["refresh"] = 4
		response["color"] = bountyColor
	}
	return response, nil
}

// findbackPetClaim is deliberately restricted to an authenticated, followed pet.
// The entire wallet/pet/progress/journal mutation occurs atomically.
func (h *Handler) findbackPetClaim(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	_, charID, err := h.characterID(ctx, "FindBackChongWu")
	if err != nil {
		return nil, err
	}
	if h.findbackPool == nil || len(args) == 0 {
		return nil, nil
	}
	mode, ok := parseIntArg(args[0])
	if !ok || (mode != 1 && mode != 2) {
		return nil, nil
	}
	day := findbackDay()
	tx, err := h.findbackPool.BeginTx(ctx.Context, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx.Context)
	var playerLevel int
	if err = tx.QueryRow(ctx.Context, `SELECT level FROM player.character_progression WHERE character_id=$1 FOR UPDATE`, charID).Scan(&playerLevel); err != nil {
		return nil, err
	}
	var money, gold int64
	if err = tx.QueryRow(ctx.Context, `SELECT money,gold FROM player.character_wallet WHERE character_id=$1 FOR UPDATE`, charID).Scan(&money, &gold); err != nil {
		return nil, err
	}
	var completed, claimed int
	if err = tx.QueryRow(ctx.Context, `SELECT completed_count,claimed_count FROM player.character_findback_days WHERE character_id=$1 AND activity_day=$2 AND activity_key='chongwu' FOR UPDATE`, charID, day).Scan(&completed, &claimed); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	missing := findbackLimits["chongwu"] - completed - claimed
	if missing <= 0 || missing > 1 {
		return nil, nil
	}
	var petID int64
	var petLevel, petPoints, petMaxPoints int
	var petExp int64
	err = tx.QueryRow(ctx.Context, `SELECT id,COALESCE(level,1),COALESCE(experience,0),attr_points,max_attr_points FROM player.character_pets WHERE character_id=$1 AND is_following=TRUE ORDER BY id LIMIT 1 FOR UPDATE`, charID).Scan(&petID, &petLevel, &petExp, &petPoints, &petMaxPoints)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if petLevel >= domainpet.MaxPetLevel || petLevel >= playerLevel+5 {
		return nil, nil
	}
	base := appactivity.FindbackProposedExperience("chongwu", playerLevel) * int64(missing)
	awarded := appactivity.FindbackAwardedExperience(base, mode)
	if awarded <= 0 {
		return nil, nil
	}
	cost := appactivity.FindbackCost("chongwu", mode, missing)
	if mode == 1 && money < cost || mode == 2 && gold < cost {
		return nil, nil
	}
	currency := "money"
	total := money - cost
	if mode == 2 {
		currency = "gold"
		total = gold - cost
	}
	if _, err = tx.Exec(ctx.Context, `UPDATE player.character_wallet SET `+currency+`=`+currency+`-$1 WHERE character_id=$2`, cost, charID); err != nil {
		return nil, err
	}
	petExp += awarded
	newLevel := petLevel
	// Do not exceed the supported level cap; preserve excess EXP at cap.
	for newLevel < domainpet.MaxPetLevel && newLevel < playerLevel+5 {
		needed := domainpet.LevelUpExperience(newLevel)
		if needed <= 0 || petExp < needed {
			break
		}
		petExp -= needed
		newLevel++
		petPoints += domainpet.AttributePointsPerLevel
		petMaxPoints += domainpet.AttributePointsPerLevel
	}
	if _, err = tx.Exec(ctx.Context, `UPDATE player.character_pets SET level=$1,experience=$2,attr_points=$3,max_attr_points=$4,updated_at=now() WHERE id=$5 AND character_id=$6`, newLevel, petExp, petPoints, petMaxPoints, petID, charID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx.Context, `UPDATE player.character_findback_days SET claimed_count=claimed_count+$1,updated_at=now() WHERE character_id=$2 AND activity_day=$3 AND activity_key='chongwu'`, missing, charID, day); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx.Context, `INSERT INTO player.character_findback_claims(character_id,activity_day,activity_key,claim_mode,count_claimed,exp_awarded,cost_currency,cost_amount) VALUES($1,$2,'chongwu',$3,$4,$5,$6,$7)`, charID, day, mode, missing, awarded, currency, cost); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx.Context); err != nil {
		return nil, err
	}
	h.sendOfflineCurrencyRefresh(ctx, charID, currency, -cost, total)
	if ctx.Connection != nil {
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(petID), "exp", fmt.Sprintf("%d", domainpet.ExperienceForLevel(newLevel)+petExp))
		if newLevel != petLevel {
			_ = ctx.Connection.SendCallback("onUpdatePet", float64(petID), "level", fmt.Sprintf("%d", newLevel))
		}
	}
	return map[string]interface{}{"type": 3, "num": completed + missing, "exp": awarded}, nil
}
