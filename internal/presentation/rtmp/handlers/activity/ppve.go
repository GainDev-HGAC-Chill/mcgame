// Open-sourced by BaoLT

// PPVE tower challenges fail closed until a catalog-backed floor opponent can enter server-owned combat.
// Pushes changePPLoadingState(cid) then onSendReplayPPVEPanel(result, floor, replayId, rankList, myRank).
// Battle result constants match GamePredef on the Flash client: BATTLE_WIN=1, BATTLE_LOSE=2, BATTLE_NOT_END=-1.
package activity

import (
	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

const (
	ppveBattleWin    = 1
	ppveBattleNotEnd = -1
)

func (h *Handler) ChallengeNextFloor(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charIDStr, characterID, err := h.characterID(ctx, "ChallengeNextFloor")
	if err != nil {
		return nil, err
	}

	h.sendActivityCallback(ctx, "changePPLoadingState", characterID)
	h.logger.Warn("ChallengeNextFloor: unsupported without authoritative floor opponent roster",
		zap.Uint32("conn_id", ctx.ConnID),
		zap.String("char_id", charIDStr),
		zap.String("reason", "missing_floor_opponent_roster"))
	h.sendActivityCallback(ctx, "onSendReplayPPVEPanel", ppveBattleNotEnd, 0, "", []interface{}{}, -1)

	return nil, nil
}

func buildPPVERankList(entries []apppet.PPVERankEntry) []interface{} {
	out := make([]interface{}, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]interface{}{
			"cid":      e.CID,
			"classId":  e.ClassID,
			"floorNum": e.FloorNum,
			"level":    e.Level,
			"name":     e.Name,
		})
	}
	return out
}
