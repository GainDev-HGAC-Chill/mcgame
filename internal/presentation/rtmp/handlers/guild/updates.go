// Open-sourced by BaoLT

package guild

import (
	"fmt"
	"strconv"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) UpdateGuildMember(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
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

	memberID, err := parseInt64Arg(request["tableId"])
	if err != nil {
		return nil, err
	}
	newRank, err := parseIntArg(request["newData"])
	if err != nil {
		return nil, err
	}

	member, err := h.guildService.GetMemberByID(ctx.Context, memberID)
	if err != nil {
		return nil, guildError(err)
	}
	if member == nil {
		// guildItem, _, guildErr := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
		// if guildErr != nil {
		// 	return nil, guildError(guildErr)
		// }
		// if guildItem != nil {
		// 	for _, candidate := range guildItem.Members {
		// 		if candidate != nil && candidate.CharacterID == memberID {
		// 			member = candidate
		// 			memberID = candidate.ID
		// 			break
		// 		}
		// 	}
		// }
		_ = ctx.Connection.SendCallback("a", msgGuildMemberNotFound)
		return nil, nil
	}

	if err := h.guildService.PromoteMember(ctx.Context, member.GuildID, characterID, member.CharacterID, newRank); err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{
		"tableId":    memberID,
		"updateType": "rank",
		"newData":    newRank,
	}

	guildItem, guildErr := h.guildService.GetGuildForClient(ctx.Context, member.GuildID)
	if guildErr == nil {
		h.broadcastGuild(guildItem, "onUpdateGuildMember", payload)
	}

	return payload, nil
}

func (h *Handler) UpdateGuildRank(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	rankDataMap, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}

	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}

	settings, err := parseGuildRankSettings(rankDataMap)
	if err != nil {
		return nil, err
	}

	if err := h.guildService.UpdateGuildRankSettings(ctx.Context, guildItem.ID, characterID, settings); err != nil {
		return nil, guildError(err)
	}

	updatedGuild, _, guildErr := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if guildErr == nil {
		h.broadcastGuildRank(updatedGuild, rankDataMap)
	}

	return map[string]interface{}{"gid": guildItem.ID, "rankData": rankDataMap}, nil
}

func (h *Handler) UpdateGuildNotice(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	guildID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	notice, _ := args[1].(string)

	if err := h.guildService.UpdateGuildInfo(ctx.Context, guildID, characterID, notice, ""); err != nil {
		return nil, guildError(err)
	}
	updatedGuild, err := h.guildService.GetGuildForClient(ctx.Context, guildID)
	if err != nil {
		return nil, guildError(err)
	}

	payload := h.guildInfoUpdateDTO(updatedGuild, 3)
	h.broadcastGuildInfo(updatedGuild, payload)

	return payload, nil
}

func (h *Handler) ChangeGuildName(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	name, _ := args[0].(string)
	if name == "" {
		return nil, pkgerrors.ErrInvalidInput
	}

	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}

	updatedGuild, err := h.guildService.UpdateGuildName(ctx.Context, guildItem.ID, characterID, name)
	if err != nil {
		return nil, guildError(err)
	}

	h.broadcastGuild(updatedGuild, "onChangeGuildName", name)
	if h.rtmpServer != nil {
		h.rtmpServer.BroadcastToAll("onChangeGuildName", name)
	}

	return name, nil
}

func (h *Handler) DelGuild(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
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

	memberIDs := make([]int64, 0, len(guildItem.Members))
	for _, member := range guildItem.Members {
		if member == nil {
			continue
		}
		memberIDs = append(memberIDs, member.CharacterID)
	}

	if err := h.guildService.DisbandGuild(ctx.Context, guildItem.ID, characterID); err != nil {
		return nil, guildError(err)
	}

	if h.rtmpServer != nil {
		h.rtmpServer.BroadcastToAll("onDelGuild", guildItem.ID)
	}
	for _, memberID := range memberIDs {
		h.syncCharacterGuildMembership(ctx.Context, memberID, nil)
		h.sendContributionState(memberID, msgGuildLeaveStateApplied)
	}

	return guildItem.ID, nil
}

func (h *Handler) GuildLevelUp(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	guildID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	updatedGuild, err := h.guildService.UpgradeGuildLevel(ctx.Context, guildID, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{"level": updatedGuild.Level}
	h.broadcastGuildProperty(updatedGuild, payload)

	return payload, nil
}

func (h *Handler) AddGuildBankSlotNum(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	guildID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	pageCount, err := h.guildService.IncreaseBankPageCount(ctx.Context, guildID, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	updatedGuild, _, guildErr := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if guildErr == nil && updatedGuild != nil {
		payload := map[string]interface{}{"bagSlotNum": pageCount}
		h.broadcastGuildProperty(updatedGuild, payload)
		return payload, nil
	}

	return map[string]interface{}{"bagSlotNum": pageCount}, nil
}

func (h *Handler) AddMaxGuildMemberNum(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	guildID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	updatedGuild, err := h.guildService.IncreaseMemberCapacity(ctx.Context, guildID, characterID)
	if err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{"memLimit": updatedGuild.MaxPopulation}
	h.broadcastGuildProperty(updatedGuild, payload)

	return payload, nil
}

func (h *Handler) ContribMoney(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
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

	silverValue, _ := request["s"]
	goldValue, _ := request["g"]
	silver, err := parseInt64Arg(silverValue)
	if err != nil {
		return nil, err
	}
	gold, err := parseInt64Arg(goldValue)
	if err != nil {
		return nil, err
	}
	if silver < 0 || gold < 0 || (silver == 0 && gold == 0) {
		return nil, pkgerrors.ErrInvalidInput
	}

	guildItem, _, err := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if err != nil {
		return nil, guildError(err)
	}
	if guildItem == nil {
		return nil, pkgerrors.ErrNotFound
	}
	if err := h.requireGuildBuildingFunction(ctx, guildItem.ID, "showDonatePanel"); err != nil {
		return nil, err
	}

	committed, err := h.guildService.ContributeMoneyAtomic(ctx.Context, guildItem.ID, characterID, silver, gold)
	if err != nil {
		return nil, guildError(err)
	}
	if silver > 0 {
		_ = ctx.Connection.SendCallback("onMinusMoney", characterID, "money", silver, committed.Character.Money)
	}
	if gold > 0 {
		_ = ctx.Connection.SendCallback("onMinusMoney", characterID, "gold", gold, committed.Character.Gold)
	}
	h.sendContributionState(characterID, "")
	switch {
	case silver > 0 && gold > 0:
		_ = ctx.Connection.SendCallback("onSystemSay", fmt.Sprintf("%s đã quyên góp %d bạc và %d vàng.", committed.CharacterName, silver, gold))
	case silver > 0:
		_ = ctx.Connection.SendCallback("onSystemSay", fmt.Sprintf("%s đã quyên góp %d bạc.", committed.CharacterName, silver))
	case gold > 0:
		_ = ctx.Connection.SendCallback("onSystemSay", fmt.Sprintf("%s đã quyên góp %d vàng.", committed.CharacterName, gold))
	}

	return map[string]interface{}{"ok": true}, nil
}

func (h *Handler) DevelopSkill(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	args = trimResponderArg(args)
	if h.gameData == nil {
		return nil, pkgerrors.ErrSystemError
	}
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
	guildID, err := parseInt64Arg(request["gid"])
	if err != nil {
		return nil, err
	}
	skillID, err := parseIntArg(request["sid"])
	if err != nil {
		return nil, err
	}

	skill := h.gameData.GetSkill(skillID)
	if skill == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	if err := h.requireGuildBuildingFunction(ctx, guildID, "showGuildSkillPanel"); err != nil {
		return nil, err
	}

	if err := h.guildService.DevelopSkill(ctx.Context, guildID, characterID, skillID); err != nil {
		return nil, guildError(err)
	}

	payload := map[string]interface{}{
		"gid":          guildID,
		"sid":          skillID,
		"skillDevData": map[string]interface{}{strconv.Itoa(skillID): 1},
	}

	guildItem, _, guildErr := h.guildService.GetGuildByMemberForClient(ctx.Context, characterID)
	if guildErr == nil && guildItem != nil {
		payload["skillDevData"] = h.filteredSkillDevData(guildItem.SkillDevData)
		h.broadcastGuild(guildItem, "onGuildSkillDeveloped", payload)
	}

	return payload, nil
}
