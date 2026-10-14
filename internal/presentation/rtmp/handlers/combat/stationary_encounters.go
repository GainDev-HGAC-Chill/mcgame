package combat

import (
	"context"
	"strconv"
	"time"

	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) RunStationaryEncounters(ctx context.Context) {
	if h.buffService == nil || h.rtmpServer == nil {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			for _, idStr := range h.rtmpServer.GetOnlineCharacterIDs() {
				if ctx.Err() != nil {
					return
				}

				conn := h.rtmpServer.GetConnectionByCharacterID(idStr)
				if conn == nil || conn.IsInBattle() {
					continue
				}

				now := time.Now()

				if conn.IsRandomEncounterOnCooldown(now) {
					continue
				}

				charID, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					continue
				}

				buffs, err := h.buffService.ListActive(ctx, charID)
				if err != nil {
					h.logger.Warn(
						"Stationary encounter: failed to load buffs",
						zap.Int64("character_id", charID),
						zap.Error(err),
					)
					continue
				}

				hasStationary := false
				hasNoEncounter := false

				for _, b := range buffs {
					switch b.BuffID {
					case 1203:
						hasStationary = true
					case 313:
						hasNoEncounter = true
					}
				}

				if !hasStationary || hasNoEncounter {
					continue
				}

				rpcCtx := &rtmp.RPCContext{
					Context:     ctx,
					ConnID:      conn.ID,
					CharacterID: idStr,
					Connection:  conn,
					Timestamp:   now,
					Method:      "stationaryEncounterTick",
				}

				// CheckBattleOnMove hiện chỉ dùng tọa độ để ghi log.
				_, err = h.CheckBattleOnMove(
					rpcCtx,
					[]interface{}{0, 0},
				)

				if err != nil {
					h.logger.Warn(
						"Stationary encounter failed",
						zap.Int64("character_id", charID),
						zap.Error(err),
					)
				}
			}
		}
	}
}
