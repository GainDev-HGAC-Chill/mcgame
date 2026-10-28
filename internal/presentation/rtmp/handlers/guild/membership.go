// Open-sourced by BaoLT

package guild

import (
	appguild "mcgame-server/internal/application/guild"
	"mcgame-server/internal/domain/guild"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func (h *Handler) AddGuild(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	guildData, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}

	guildName, _ := guildData["name"].(string)
	if guildName == "" {
		return nil, pkgerrors.ErrInvalidInput
	}

	committed, err := h.guildService.CreateGuildAtomic(ctx.Context, characterID, guildName)
	if err != nil {
		switch guildError(err) {
		case appguild.ErrInsufficientLevel:
			_ = ctx.Connection.SendCallback("a", msgGuildCreateInsufficientLevel)
			return nil, nil
		case appguild.ErrInsufficientFunds:
			_ = ctx.Connection.SendCallback("a", msgGuildCreateInsufficientFunds)
			return nil, nil
		case appguild.ErrGuildNameTaken:
			_ = ctx.Connection.SendCallback("a", msgGuildNameTaken)
			return nil, nil
		case appguild.ErrAlreadyInGuild:
			_ = ctx.Connection.SendCallback("a", msgAlreadyInGuild)
			return nil, nil
		default:
			return nil, guildError(err)
		}
	}

	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	response := h.guildDTO(guildItem)
	if err := ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"money": committed.Leader.Money, "moneyBind": committed.Leader.MoneyBind,
		"guildContrib": committed.Leader.GuildContrib, "donateContrib": committed.Leader.DonateContrib,
	}); err != nil {
		h.logger.Warn("AddGuild: failed to send onUPP", zap.Error(err))
	}
	if h.rtmpServer != nil {
		h.rtmpServer.BroadcastToAll("onAddGuild", response)
	}

	panelResponse, _, _, buildErr := h.buildGuildPanelResponse(ctx.Context, characterID)
	if buildErr == nil {
		_ = ctx.Connection.SendCallback("onInitViewGuildP", panelResponse)
	}
	_ = ctx.Connection.SendCallback("a", msgGuildCreateGuide)
	if committed.Leader.GuildContrib > 0 || committed.Leader.DonateContrib > 0 {
		_ = ctx.Connection.SendCallback("a", msgGuildRestoreApplied)
	}

	return response, nil
}

func (h *Handler) AddGuildMember(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	request, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}

	guildIDValue, ok := request["gid"]
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	guildID, err := parseInt64Arg(guildIDValue)
	if err != nil {
		return nil, err
	}

	application, err := h.guildService.ApplyToGuild(ctx.Context, guildID, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	response := h.applicationDTO(application)
	_ = ctx.Connection.SendCallback("onAddGuildMember", response)

	guildItem, guildErr := h.guildService.GetGuildForClient(ctx.Context, guildID)
	if guildErr == nil {
		h.broadcastGuild(guildItem, "onAddGuildMember", response)
	}

	return response, nil
}

func (h *Handler) QuitGuild(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	memberID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	application, appGuild, appErr := h.guildService.GetPendingApplicationByCharacterID(ctx.Context, characterID)
	if appErr != nil {
		return nil, guildError(appErr)
	}
	if application != nil && application.ID == memberID {
		if _, err := h.guildService.CancelApplication(ctx.Context, application.ID, characterID); err != nil {
			return nil, guildError(err)
		}
		payload := map[string]interface{}{
			"id":   application.ID,
			"gid":  application.GuildID,
			"cid":  application.CharacterID,
			"name": application.CharacterName,
			"type": 4,
		}
		_ = ctx.Connection.SendCallback("onDelGuildMember", payload)
		if appGuild != nil {
			h.broadcastGuild(appGuild, "onDelGuildMember", payload)
		}
		return payload, nil
	}
	member, err := h.guildService.GetMemberByID(ctx.Context, memberID)
	if err != nil {
		return nil, guildError(err)
	}
	if member == nil {
		return nil, pkgerrors.ErrNotFound
	}

	if member.CharacterID != characterID {
		return nil, pkgerrors.ErrUnauthorized
	}

	if member.Rank == guild.RankLeader {
		return nil, guildError(appguild.ErrNotGuildLeader)
	}

	if err := h.guildService.LeaveMember(ctx.Context, member.GuildID, member.CharacterID); err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{
		"id":   member.ID,
		"gid":  member.GuildID,
		"cid":  member.CharacterID,
		"name": member.CharacterName,
		"type": 2,
	}

	guildItem, guildErr := h.guildService.GetGuildForClient(ctx.Context, member.GuildID)
	if guildErr == nil {
		h.broadcastGuild(guildItem, "onDelGuildMember", payload)
	}
	h.sendToCharacter(member.CharacterID, "onDelGuildMember", payload)
	h.sendContributionState(member.CharacterID, msgGuildLeaveStateApplied)

	return payload, nil
}

func (h *Handler) ConfirmGuildApply(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	request, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	applicationID, err := parseInt64Arg(request["tableId"])
	if err != nil {
		return nil, err
	}

	application, member, err := h.guildService.ApproveApplication(ctx.Context, applicationID, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	payload := h.guildApprovalPayload(application, member)

	guildItem, guildErr := h.guildService.GetGuildForClient(ctx.Context, application.GuildID)
	if guildErr == nil {
		h.notifyGuildApproval(guildItem, application, member, payload)
		panelResponse, _, _, panelErr := h.buildGuildPanelResponse(ctx.Context, characterID)
		if panelErr != nil {
			h.logger.Warn("ConfirmGuildApply: failed to rebuild approver panel", zap.Error(panelErr))
		} else if sendErr := h.sendGuildPanelRefresh(ctx.Connection, panelResponse); sendErr != nil {
			h.logger.Warn("ConfirmGuildApply: failed to refresh approver panel", zap.Error(sendErr))
		}
	} else {
		h.sendGuildApprovalToApplicant(application.CharacterID, member, payload)
		h.logger.Warn("ConfirmGuildApply: failed to load committed guild", zap.Error(guildErr))
	}

	return payload, nil
}

func (h *Handler) RefuseGuildMember(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	applicationID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	application, err := h.guildService.RefuseApplication(ctx.Context, applicationID, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{
		"id":   application.ID,
		"gid":  application.GuildID,
		"cid":  application.CharacterID,
		"name": application.CharacterName,
		"type": 3,
	}

	guildItem, guildErr := h.guildService.GetGuildForClient(ctx.Context, application.GuildID)
	if guildErr == nil {
		h.broadcastGuild(guildItem, "onDelGuildMember", payload)
	}
	h.sendToCharacter(application.CharacterID, "onDelGuildMember", payload)

	return payload, nil
}

func (h *Handler) KickGuildMember(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	memberID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	member, err := h.guildService.GetMemberByID(ctx.Context, memberID)
	if err != nil {
		return nil, guildError(err)
	}
	if member == nil {
		return nil, pkgerrors.ErrNotFound
	}

	if err := h.guildService.KickMember(ctx.Context, member.GuildID, characterID, member.CharacterID); err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{
		"id":   member.ID,
		"gid":  member.GuildID,
		"cid":  member.CharacterID,
		"name": member.CharacterName,
		"type": 1,
	}

	guildItem, guildErr := h.guildService.GetGuildForClient(ctx.Context, member.GuildID)
	if guildErr == nil {
		h.broadcastGuild(guildItem, "onDelGuildMember", payload)
	}
	h.sendToCharacter(member.CharacterID, "onDelGuildMember", payload)
	h.sendContributionState(member.CharacterID, msgGuildLeaveStateApplied)

	return payload, nil
}

func (h *Handler) DemiseTo(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	newLeaderMemberID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	guildItem, selfMember, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil || selfMember == nil {
		return nil, pkgerrors.ErrNotFound
	}

	newLeaderMember, err := h.guildService.GetMemberByID(ctx.Context, newLeaderMemberID)
	if err != nil {
		return nil, guildError(err)
	}
	if newLeaderMember == nil || newLeaderMember.GuildID != guildItem.ID {
		return nil, pkgerrors.ErrNotFound
	}
	newLeaderID := newLeaderMember.CharacterID
	updatedGuild, err := h.guildService.TransferLeadership(ctx.Context, guildItem.ID, characterID, newLeaderID)
	if err != nil {
		return nil, guildError(err)
	}

	leaderName := ""
	for _, member := range updatedGuild.Members {
		if member != nil && member.CharacterID == newLeaderID {
			leaderName = member.CharacterName
			break
		}
	}

	payload := map[string]interface{}{
		"gid":    updatedGuild.ID,
		"oldCid": characterID,
		"newCid": newLeaderID,
		"ln":     leaderName,
	}

	h.broadcastGuild(updatedGuild, "onDemise", payload)
	h.broadcastGuildInfo(updatedGuild, h.guildDTO(updatedGuild))

	return payload, nil
}
