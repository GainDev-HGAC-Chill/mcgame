// Open-sourced by BaoLT

package guild

import (
	"strings"

	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	guildBuildPanelName    = "Công trình bang hội"
	guildBuildPanelContent = "Mọi người hãy đến xây dựng"
	guildWarehousePanelID  = 5
	guildBuildingType      = 1
	guildExtendBuildType   = 3
	guildExtendTemplateID  = 1
)

func (h *Handler) ClickBuild(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	if len(args) < 2 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	buildID, err := parseIntArg(args[0])
	if err != nil {
		return nil, err
	}
	buildType, err := parseIntArg(args[1])
	if err != nil {
		return nil, err
	}

	if h.guildService == nil {
		return nil, pkgerrors.ErrSystemError
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
	callback, templateID, err := resolveGuildBuildClick(int64(buildID), buildType, buildings)
	if err != nil {
		return nil, err
	}
	ctx.Connection.SetLastGuildBuildContext(buildID, templateID)
	if callback != "" {
		if err := ctx.Connection.SendCallbackSync(callback, buildID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": buildID}, nil
	}
	options := h.buildFuncOptions(buildID, templateID, false)
	if err := ctx.Connection.SendCallbackSync("onList", buildID, guildBuildPanelName, guildBuildPanelContent, options); err != nil {
		return nil, err
	}
	return options, nil
}

func resolveGuildBuildClick(buildID int64, buildType int, buildings []*domainguild.BuildingState) (string, int, error) {
	for _, building := range buildings {
		if building == nil || building.ID != buildID {
			continue
		}
		if buildType != guildBuildingType {
			return "", 0, pkgerrors.ErrInvalidInput
		}
		if building.Pending() {
			return "onShowBuildProgress", building.PendingTemplateID, nil
		}
		return "", building.TemplateID, nil
	}
	if buildType == guildExtendBuildType {
		return "", guildExtendTemplateID, nil
	}
	return "", 0, pkgerrors.ErrNotFound
}

func (h *Handler) ExecBuildFunc(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	funcName, ok := args[0].(string)
	if !ok || funcName == "" {
		return nil, pkgerrors.ErrInvalidInput
	}

	buildID := 0
	buildType := 0
	if len(args) > 1 {
		if parsed, err := parseIntArg(args[1]); err == nil {
			buildID = parsed
		}
	}
	if len(args) > 2 {
		if parsed, err := parseIntArg(args[2]); err == nil {
			buildType = parsed
		}
	}
	if buildID == 0 || buildType == 0 {
		lastBuildID, lastBuildType := ctx.Connection.GetLastGuildBuildContext()
		if buildID == 0 {
			buildID = lastBuildID
		}
		if buildType == 0 {
			buildType = lastBuildType
		}
	}
	if buildID > 0 && buildType > 0 {
		ctx.Connection.SetLastGuildBuildContext(buildID, buildType)
	}

	switch funcName {
	case "extInfo":
		if buildID == 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		if err := ctx.Connection.SendCallback("onShowExtInfo", buildID); err != nil {
			return nil, err
		}
		return nil, nil
	case "showGuildWarehouse":
		if buildID == 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		if err := h.requireCompletedBuildingFunction(ctx, characterID, funcName); err != nil {
			return nil, err
		}
		if err := ctx.Connection.SendCallback("onShowGuildWarehouse", guildWarehousePanelID); err != nil {
			return nil, err
		}
		return nil, nil
	case "showDonatePanel":
		if buildID == 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		if err := h.requireCompletedBuildingFunction(ctx, characterID, funcName); err != nil {
			return nil, err
		}
		if err := ctx.Connection.SendCallback("onShowDonatePanel", guildWarehousePanelID); err != nil {
			return nil, err
		}
		return nil, nil
	case "showGuildSkillPanel":
		if err := h.requireCompletedBuildingFunction(ctx, characterID, funcName); err != nil {
			return nil, err
		}
		payload, err := h.GetGuildPrivateSkillData(ctx, nil)
		if err != nil {
			return nil, err
		}
		if err := ctx.Connection.SendCallback("onShowGuildSkillPanel", payload); err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, pkgerrors.ErrInvalidInput
	}
}

func (h *Handler) requireCompletedBuildingFunction(ctx *rtmp.RPCContext, characterID int64, functionName string) error {
	if h.guildService == nil {
		return pkgerrors.ErrSystemError
	}
	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return guildError(err)
	}
	if guildItem == nil {
		return pkgerrors.ErrNotFound
	}
	return h.requireGuildBuildingFunction(ctx, guildItem.ID, functionName)
}

func (h *Handler) requireGuildBuildingFunction(ctx *rtmp.RPCContext, guildID int64, functionName string) error {
	available, err := h.guildService.BuildingFunctionAvailable(ctx.Context, guildID, functionName)
	if err != nil {
		return guildError(err)
	}
	if !available {
		return pkgerrors.ErrNotFound
	}
	return nil
}

func (h *Handler) buildFuncOptions(buildID int, templateID int, npcScript bool) []map[string]interface{} {
	options := make([]map[string]interface{}, 0, 4)
	if h.gameData == nil {
		return options
	}
	template := h.gameData.GetBuilding(templateID)
	if template == nil {
		return options
	}
	if strings.Contains(template.FuncScript, "extInfo") {
		funcName := "extInfo"
		if npcScript {
			funcName = "buildInfo"
		}
		options = append(options, map[string]interface{}{
			"label": "Thông tin công trình",
			"func":  funcName,
			"bid":   buildID,
		})
	}
	if strings.Contains(template.FuncScript, "showGuildWarehouse") {
		options = append(options, map[string]interface{}{
			"label": "Kho bang",
			"func":  "showGuildWarehouse",
			"bid":   buildID,
		})
	}
	if strings.Contains(template.FuncScript, "showDonatePanel") {
		options = append(options, map[string]interface{}{
			"label": "Quyên góp",
			"func":  "showDonatePanel",
			"bid":   buildID,
		})
	}
	if strings.Contains(template.FuncScript, "showGuildSkillPanel") {
		options = append(options, map[string]interface{}{
			"label": "Kỹ năng bang",
			"func":  "showGuildSkillPanel",
			"bid":   buildID,
		})
	}
	return options
}
