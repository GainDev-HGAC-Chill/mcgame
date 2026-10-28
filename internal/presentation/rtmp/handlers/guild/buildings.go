// Open-sourced by BaoLT

package guild

import (
	"sort"
	"strconv"

	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	guildMapID            = 49
	guildTransportTargetX = 1800
	guildTransportTargetY = 1360
)

func (h *Handler) CreateGuildBuildings(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if h.guildService == nil || h.gameData == nil {
		return []interface{}{}, nil
	}

	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return []interface{}{}, nil
	}
	mapID, _ := ctx.Connection.GetSceneInfo()
	if mapID != guildMapID {
		return []interface{}{}, nil
	}

	positions := h.gameData.GetExtendPositionsByMapID(guildMapID)
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].ID < positions[j].ID
	})

	buildings, err := h.guildService.ListGuildBuildings(ctx.Context, guildItem.ID)
	if err != nil {
		return nil, guildError(err)
	}
	occupied := make(map[int]interface{}, len(buildings))
	for _, building := range buildings {
		if building == nil {
			continue
		}
		position := h.guildBuildingPosition(building.ExtendID)
		if position == nil {
			return nil, pkgerrors.ErrSystemError
		}
		if building.Pending() {
			occupied[building.ExtendID] = h.pendingGuildBuildingPayload(guildItem.ID, building, position)
		} else {
			occupied[building.ExtendID] = h.completedGuildBuildingPayload(guildItem.ID, building, position, guildItem.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}
	result := make([]interface{}, 0, len(positions))
	for _, pos := range positions {
		if pos == nil {
			continue
		}

		if payload, ok := occupied[int(pos.ID)]; ok {
			result = append(result, payload)
			continue
		}

		result = append(result, emptyGuildBuildingPayload(pos))
	}

	if err := ctx.Connection.SendCallback("onCreateGuildBuildings", result); err != nil {
		return nil, err
	}
	for _, building := range buildings {
		if building == nil || !building.Pending() {
			continue
		}
		if err := ctx.Connection.SendCallback("setBuildNpcState", map[string]interface{}{"state": 1, "eid": building.ExtendID}); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func emptyGuildBuildingPayload(position *models.ExtendPositionTemplate) map[string]interface{} {
	return map[string]interface{}{
		"posX":      strconv.Itoa(int(position.PosX)),
		"layer":     "0",
		"tid":       strconv.Itoa(int(position.Tid)),
		"id":        strconv.Itoa(int(position.ID)),
		"mid":       strconv.Itoa(int(position.Mid)),
		"posY":      strconv.Itoa(int(position.PosY)),
		"posDir":    strconv.Itoa(int(position.PosDir)),
		"buildType": strconv.Itoa(int(position.BuildType)),
	}
}
