package combat

import (
	"fmt"
	"strconv"

	appcombat "mcgame-server/internal/application/combat"
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) SetSTHBattleResultHandler(callback func(*rtmp.RPCContext, int64, string, bool) error) {
	h.sthBattleResultHandler = callback
}

// StartSTHBattle uses real creature templates and the standard battle lifecycle.
func (h *Handler) StartSTHBattle(ctx *rtmp.RPCContext, npcID int) (string, error) {
	if h.gameData == nil {
		return "", fmt.Errorf("STH: game data unavailable")
	}
	cid, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return "", err
	}
	char, err := h.charService.GetByID(ctx.Context, cid)
	if err != nil {
		return "", err
	}
	rows := h.gameData.GetNpcCreaturesByNid(npcID)
	if len(rows) == 0 {
		return "", fmt.Errorf("STH NPC %d has no linked creatures", npcID)
	}
	templates := make([]appcombat.NPCTemplateData, 0, len(rows))
	for _, row := range rows {
		creatureID := int(row.Cid)
		creature := h.gameData.GetCreature(creatureID)
		if creature == nil {
			return "", fmt.Errorf("STH creature %d missing", creatureID)
		}
		tmpl := appcombat.CalculateCreatureStats(creature, char.Level)
		tmpl.ID = creatureID
		tmpl.Exp = int(row.Exp)
		tmpl.ExpMulti = row.ExpMulti
		tmpl.BossFlag = int(row.BossFlag)
		templates = append(templates, tmpl)
	}
	mapID, _ := ctx.Connection.GetSceneInfo()
	battle, err := h.combatService.StartPVEBattleWithMultipleEnemies(ctx.Context, cid, mapID, ctx.Connection.GetChannelID(), templates, ctx.Connection.IsPlayerInFront())
	if err != nil {
		return "", err
	}
	h.notifyBattleStarted(battle)
	return battle.ID, nil
}
