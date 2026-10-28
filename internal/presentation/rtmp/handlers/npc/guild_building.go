// Open-sourced by BaoLT

package npc

import (
	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/infrastructure/rtmp"
)

const (
	guildBuildConstructNPCID    = 1098
	guildBuildInfoNPCID         = 1099
	guildBuildWarehouseNPCID    = 1102
	guildBuildConstructID       = 10
	guildBuildConstructType     = 1
	guildBuildInfoID            = 169
	guildBuildWarehouseID       = 5
	guildBuildInfoType          = 4
	guildBuildWarehouseType     = 5
	guildBuildPrompt            = "Hãy gọi bang chủ đến mở nhiệm vụ bang hội đi."
	guildBuildListTitle         = "Công trình bang hội"
	guildBuildListContent       = "Mọi người hãy đến xây dựng"
	guildBuildInfoFallbackID    = 169
	guildBuildWarehouseFallback = 5
)

func (h *Handler) tryHandleGuildBuildingNPCClick(ctx *rtmp.RPCContext, characterID int64, npcID int) (bool, error) {
	switch npcID {
	case guildBuildConstructNPCID:
		building, err := h.guildBuildingAtExtension(ctx, characterID, guildBuildConstructID)
		if err != nil {
			return true, err
		}
		if building != nil {
			ctx.Connection.SetLastGuildBuildContext(int(building.ID), buildingClientTemplateID(building))
			callback := "onShowBuildInfo"
			if building.Pending() {
				callback = "onShowBuildProgress"
			}
			if err := ctx.Connection.SendCallbackSync(callback, int(building.ID)); err != nil {
				return true, err
			}
			return true, nil
		}
		ctx.Connection.SetLastGuildBuildContext(guildBuildConstructID, guildBuildConstructType)
		options := []map[string]interface{}{
			{"label": "Bắt đầu xây", "func": "startConstruction"},
		}
		if err := ctx.Connection.SendCallbackSync("onList", npcID, guildBuildListTitle, guildBuildListContent, options); err != nil {
			return true, err
		}
		return true, nil
	case guildBuildInfoNPCID:
		ctx.Connection.SetLastGuildBuildContext(guildBuildInfoID, guildBuildInfoType)
		_ = ctx.Connection.SendCallback("onNpcMsg", npcID, guildBuildPrompt)
		options := []map[string]interface{}{
			{"label": "Thông tin công trình", "func": "buildInfo"},
		}
		if err := ctx.Connection.SendCallbackSync("onList", npcID, guildBuildListTitle, guildBuildListContent, options); err != nil {
			return true, err
		}
		return true, nil
	case guildBuildWarehouseNPCID:
		ctx.Connection.SetLastGuildBuildContext(guildBuildWarehouseID, guildBuildWarehouseType)
		_ = ctx.Connection.SendCallback("onNpcMsg", npcID, guildBuildPrompt)
		options := []map[string]interface{}{{"label": "Thông tin công trình", "func": "buildInfo"}}
		available, err := h.guildBuildingFunctionAvailable(ctx, characterID, "showGuildWarehouse")
		if err != nil {
			return true, err
		}
		if available {
			options = append([]map[string]interface{}{
				{"label": "Kho bang", "func": "showGuildWarehouse"},
				{"label": "Quyên góp", "func": "showDonatePanel"},
			}, options...)
		}
		if err := ctx.Connection.SendCallbackSync("onList", npcID, guildBuildListTitle, guildBuildListContent, options); err != nil {
			return true, err
		}
		return true, nil
	default:
		return false, nil
	}
}

func (h *Handler) guildBuildingFunctionAvailable(ctx *rtmp.RPCContext, characterID int64, functionName string) (bool, error) {
	if h.guildService == nil {
		return false, nil
	}
	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil || guildItem == nil {
		return false, err
	}
	return h.guildService.BuildingFunctionAvailable(ctx.Context, guildItem.ID, functionName)
}

func (h *Handler) guildBuildingAtExtension(ctx *rtmp.RPCContext, characterID int64, extendID int) (*domainguild.BuildingState, error) {
	if h.guildService == nil {
		return nil, nil
	}
	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil || guildItem == nil {
		return nil, err
	}
	buildings, err := h.guildService.ListGuildBuildings(ctx.Context, guildItem.ID)
	if err != nil {
		return nil, err
	}
	return findGuildBuildingAtExtension(buildings, extendID), nil
}

func findGuildBuildingAtExtension(buildings []*domainguild.BuildingState, extendID int) *domainguild.BuildingState {
	for _, building := range buildings {
		if building != nil && building.ExtendID == extendID {
			return building
		}
	}
	return nil
}

func buildingClientTemplateID(building *domainguild.BuildingState) int {
	if building == nil {
		return 0
	}
	if building.Pending() {
		return building.PendingTemplateID
	}
	return building.TemplateID
}

func guildBuildCallbackID(conn *rtmp.Connection, fallback int) int {
	buildID, _ := conn.GetLastGuildBuildContext()
	if buildID <= 0 {
		return fallback
	}
	return buildID
}
