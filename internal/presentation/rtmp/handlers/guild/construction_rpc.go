// Guild construction RPCs use persisted building commands before callbacks.
package guild

import (
	"fmt"
	"strconv"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	constructionPlaceholderTID = 10
	constructionActorType      = 2
)

func (h *Handler) StartConstruction(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	if len(args) < 1 {
		buildID, _ := ctx.Connection.GetLastGuildBuildContext()
		if buildID <= 0 {
			return nil, pkgerrors.ErrInvalidArgs
		}
		return buildID, nil
	}
	buildID, err := parseIntArg(args[0])
	if err != nil {
		return nil, err
	}
	return buildID, nil
}

func (h *Handler) ConstructBuild(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 4 || h.guildService == nil || h.gameData == nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	sourceID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	targetTID, err := parseIntArg(args[1])
	if err != nil {
		return nil, err
	}
	mode, err := parseIntArg(args[2])
	if err != nil {
		return nil, err
	}
	mapID, err := parseIntArg(args[3])
	if err != nil {
		return nil, err
	}
	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}
	state, err := h.guildService.StartGuildBuilding(ctx.Context, guildItem.ID, characterID, sourceID, targetTID, mode, mapID)
	if err != nil {
		return nil, guildError(err)
	}
	position := h.guildBuildingPosition(state.ExtendID)
	buildTpl := h.gameData.GetBuilding(targetTID)
	if position == nil || buildTpl == nil {
		return nil, pkgerrors.ErrSystemError
	}
	ctx.Connection.SetLastGuildBuildContext(int(state.ID), targetTID)
	payload := h.pendingGuildBuildingPayload(guildItem.ID, state, position)
	if err := ctx.Connection.SendCallback("onCreateBuild", payload); err != nil {
		return nil, err
	}
	h.broadcastConstructionCallback(ctx.Connection, "setBuildNpcState", map[string]interface{}{"state": 1, "eid": state.ExtendID})
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	h.broadcastGuild(guildItem, "onSay", h.guildConstructionChatPayload(char.ID, char.Name, h.guildConstructionGuildPayload(guildItem), fmt.Sprintf("Bắt đầu xây dựng %s, các thành viên ai có tiền góp tiền, ai có sức góp sức.", buildTpl.Name), ctx.Connection.GetChannelID()))
	return payload, nil
}

func (h *Handler) FinishCreate(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.finishConstruction(ctx, args, appguild.BuildingCreateMode)
}

func (h *Handler) FinishUpgrade(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.finishConstruction(ctx, args, appguild.BuildingUpgradeMode)
}

func (h *Handler) ResetConstruction(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 || h.guildService == nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	buildingID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}
	buildings, err := h.guildService.ListGuildBuildings(ctx.Context, guildItem.ID)
	if err != nil {
		return nil, guildError(err)
	}
	previous := handlerGuildBuildingByID(buildings, buildingID)
	if previous == nil || !previous.Pending() {
		return nil, pkgerrors.ErrNotFound
	}
	state, err := h.guildService.ResetGuildBuilding(ctx.Context, guildItem.ID, characterID, buildingID)
	if err != nil {
		return nil, guildError(err)
	}
	callback, payload, err := h.guildResetCallback(previous, state)
	if err != nil {
		return nil, err
	}
	if err := ctx.Connection.SendCallback(callback, payload); err != nil {
		return nil, err
	}
	h.broadcastConstructionCallback(ctx.Connection, "setBuildNpcState", map[string]interface{}{"state": 0, "eid": state.ExtendID})
	return map[string]interface{}{"success": true}, nil
}

func handlerGuildBuildingByID(buildings []*domainguild.BuildingState, buildingID int64) *domainguild.BuildingState {
	for _, building := range buildings {
		if building != nil && building.ID == buildingID {
			return building
		}
	}
	return nil
}

func (h *Handler) guildResetCallback(previous, committed *domainguild.BuildingState) (string, map[string]interface{}, error) {
	if previous == nil || committed == nil {
		return "", nil, pkgerrors.ErrSystemError
	}
	var position *models.ExtendPositionTemplate
	if previous.Mode == appguild.BuildingCreateMode {
		position = h.guildBuildingPosition(committed.ExtendID)
	}
	return resetGuildBuildingCallback(previous, committed, position)
}

func resetGuildBuildingCallback(previous, committed *domainguild.BuildingState, position *models.ExtendPositionTemplate) (string, map[string]interface{}, error) {
	if previous == nil || committed == nil {
		return "", nil, pkgerrors.ErrSystemError
	}
	if previous.Mode != appguild.BuildingCreateMode {
		return "onCancelUpgrade", map[string]interface{}{"bid": committed.ID}, nil
	}
	if position == nil {
		return "", nil, pkgerrors.ErrSystemError
	}
	return "onCancelCreate", map[string]interface{}{"bid": committed.ID, "ext": emptyGuildBuildingPayload(position)}, nil
}

func (h *Handler) finishConstruction(ctx *rtmp.RPCContext, args []interface{}, expectedMode int) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 || h.guildService == nil || h.gameData == nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	buildingID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}
	committed, err := h.guildService.FinishGuildBuilding(ctx.Context, guildItem.ID, characterID, buildingID, expectedMode)
	if err != nil {
		return nil, guildError(err)
	}
	state := committed.Building
	position := h.guildBuildingPosition(state.ExtendID)
	buildTpl := h.gameData.GetBuilding(state.TemplateID)
	if position == nil || buildTpl == nil {
		return nil, pkgerrors.ErrSystemError
	}
	propertyPayload := h.guildConstructionGuildPayload(committed.Guild)
	h.broadcastGuildProperty(committed.Guild, propertyPayload)
	reloadPayload := h.completedGuildBuildingPayload(committed.Guild.ID, state, position, committed.Guild.CreatedAt.Format("2006-01-02 15:04:05"))
	h.broadcastGuild(committed.Guild, "reloadBuild", reloadPayload)
	h.broadcastConstructionCallback(ctx.Connection, "setBuildNpcState", map[string]interface{}{"state": 0, "eid": state.ExtendID})
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	message := fmt.Sprintf("Bang hội xây dựng thành công %s", buildTpl.Name)
	if expectedMode == appguild.BuildingUpgradeMode {
		message = fmt.Sprintf("Bang hội nâng cấp thành công %s", buildTpl.Name)
	}
	h.broadcastGuild(committed.Guild, "onSay", h.guildConstructionChatPayload(char.ID, char.Name, propertyPayload, message, ctx.Connection.GetChannelID()))
	return reloadPayload, nil
}

func (h *Handler) guildBuildingPosition(extendID int) *models.ExtendPositionTemplate {
	if h.gameData == nil {
		return nil
	}
	for _, position := range h.gameData.GetExtendPositionsByMapID(appguild.GuildBuildingMapID) {
		if position != nil && int(position.ID) == extendID {
			return position
		}
	}
	return nil
}

func (h *Handler) pendingGuildBuildingPayload(guildID int64, state *domainguild.BuildingState, position *models.ExtendPositionTemplate) map[string]interface{} {
	return map[string]interface{}{
		"extendId": state.ExtendID, "buildState": strconv.Itoa(state.Mode), "mid": strconv.Itoa(appguild.GuildBuildingMapID),
		"posY": strconv.Itoa(int(position.PosY)), "gid": guildID, "id": state.ID, "layer": 1,
		"posX": strconv.Itoa(int(position.PosX)), "tid": constructionPlaceholderTID, "posDir": strconv.Itoa(int(position.PosDir)),
	}
}

func (h *Handler) completedGuildBuildingPayload(guildID int64, state *domainguild.BuildingState, position *models.ExtendPositionTemplate, createdAt string) map[string]interface{} {
	return map[string]interface{}{
		"targetId": nil, "posX": strconv.Itoa(int(position.PosX)), "layer": "1", "tid": strconv.Itoa(state.TemplateID),
		"buildState": "", "posY": strconv.Itoa(int(position.PosY)), "posDir": strconv.Itoa(int(position.PosDir)),
		"permission": "", "cid": nil, "time": createdAt, "extendId": strconv.Itoa(state.ExtendID),
		"id": strconv.FormatInt(state.ID, 10), "mid": strconv.Itoa(appguild.GuildBuildingMapID), "gid": strconv.FormatInt(guildID, 10),
	}
}

func (h *Handler) guildConstructionGuildPayload(guildItem *domainguild.Guild) map[string]interface{} {
	payload := h.guildInfoUpdateDTO(guildItem, 0)
	delete(payload, "updateType")
	return payload
}

func (h *Handler) guildConstructionChatPayload(characterID int64, name string, guildPayload map[string]interface{}, message string, channelID int) map[string]interface{} {
	return map[string]interface{}{"id": characterID, "name": name, "guildName": guildPayload, "type": constructionActorType, "channelId": channelID, "pmLevel": 0, "footle": false, "msg": message}
}

func (h *Handler) broadcastConstructionCallback(conn *rtmp.Connection, method string, payload map[string]interface{}) {
	if h.rtmpServer != nil {
		h.rtmpServer.BroadcastToAll(method, payload)
		return
	}
	_ = conn.SendCallback(method, payload)
}
