// Open-sourced by BaoLT

package petarena

import (
	apppetarena "mcgame-server/internal/application/petarena"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func (h *Handler) GetPetArenaData(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	enemies, err := h.arenaService.GetEnemies(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onGetEnemies", enemies)
	h.logger.Debug("Pet arena enemies retrieved", zap.Int64("character_id", characterID))
	return enemies, nil
}

func (h *Handler) GetPetArenaActivityInfo(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.arenaService.GetActivityInfo(ctx.Context), nil
}

func (h *Handler) GetPetArenaDataActivity(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	enemies, err := h.arenaService.GetEnemies(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("getEnemiesActivity", enemies)
	return enemies, nil
}

func (h *Handler) GetPetArenaRank(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	payload, err := h.arenaService.GetArenaRankPayload(ctx.Context)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onPetArenaRank", payload)
	return payload, nil
}

func (h *Handler) GetPetRankAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	rewards, err := h.arenaService.ClaimSeasonReward(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet arena rank award claimed",
		zap.Int64("character_id", characterID))

	return map[string]interface{}{
		"success": true,
		"rewards": rewards,
	}, nil
}

func (h *Handler) GetPetArenaSysMsg(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return map[string]interface{}{
		"messages": h.arenaService.GetSystemMessages(ctx.Context),
	}, nil
}

func (h *Handler) GetPetArenaConf(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := h.arenaService.GetPetFightConfig(ctx.Context, characterID, apppetarena.ConfigContextArena)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	response := make(map[string]interface{}, len(payload)+6)
	for key, value := range payload {
		response[key] = value
	}
	config := h.arenaService.GetConfig()
	response["season"] = config.Season
	response["maxTickets"] = config.MaxTickets
	response["ticketRefreshHours"] = config.TicketRefreshHours
	response["baseRating"] = config.BaseRating
	response["ratingKFactor"] = config.RatingKFactor
	response["winStreakBonusStart"] = config.WinStreakBonusStart
	if err := ctx.Connection.SendCallback("onGetPetConf", response); err != nil {
		h.logger.Warn("Failed to send arena onGetPetConf callback", zap.Error(err))
	}
	return response, nil
}

func (h *Handler) GetPetConf(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	payload, err := h.arenaService.GetPetFightConfig(ctx.Context, characterID, apppetarena.ConfigContextFarm)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	if err := ctx.Connection.SendCallback("onGetPetConf", payload); err != nil {
		h.logger.Warn("Failed to send onGetPetConf callback", zap.Error(err))
	}

	return payload, nil
}

func (h *Handler) PetFightConf(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := ensureArgs(args, 1); err != nil {
		return nil, err
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	confData, ok := normalizeStringMap(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}

	configContext, isXiulianshi := petFightConfigContext(args)
	payload := map[string]interface{}{
		"conf1": confData,
	}
	if err := h.arenaService.SavePetFightConfig(ctx.Context, characterID, configContext, payload); err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	callback := "onSubPetConf"
	if isXiulianshi {
		callback = "onSubPetConfMT"
	}
	if err := ctx.Connection.SendCallback(callback, true); err != nil {
		h.logger.Warn("Failed to send pet fight config callback", zap.Error(err))
	}

	return map[string]interface{}{"success": true}, nil
}

func petFightConfigContext(args []interface{}) (string, bool) {
	isXiulianshi := false
	if len(args) >= 3 {
		switch typed := args[2].(type) {
		case bool:
			isXiulianshi = typed
		case float64:
			isXiulianshi = typed != 0
		}
	}
	if isXiulianshi {
		return apppetarena.ConfigContextHMTXLS, true
	}
	return apppetarena.ConfigContextFarm, false
}

func (h *Handler) PetArenaTicket(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if err := ensureArgs(args, 2); err != nil {
		return nil, err
	}
	teamName, _ := args[0].(string)
	confData, ok := normalizeStringMap(args[1])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}

	if _, err := h.arenaService.RegisterArenaEntry(ctx.Context, characterID, teamName, confData); err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	enemies, err := h.arenaService.GetEnemies(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	_ = ctx.Connection.SendCallback("onGetEnemies", enemies)
	_ = ctx.Connection.SendCallback("a", "Báo danh thành công")
	return enemies, nil
}

func (h *Handler) GetLastPetArenaRank(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}

	rank, rating, err := h.arenaService.GetLastSeasonRank(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	return map[string]interface{}{
		"rank":   rank,
		"rating": rating,
	}, nil
}

func ensureArgs(args []interface{}, size int) error {
	if len(args) < size {
		return pkgerrors.ErrInvalidArgs
	}
	return nil
}
