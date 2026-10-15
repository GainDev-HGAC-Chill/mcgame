// Open-sourced by BaoLT

package item

import (
	"mcgame-server/internal/infrastructure/rtmp"
)

const secretPassageMapID = 140106

func isSecretPassageBook(templateID int) bool {
	return templateID == 128 || templateID == 2137
}

func (h *Handler) canUseSecretPassageBook(ctx *rtmp.RPCContext, characterID int64) (bool, error) {
	if ctx == nil || ctx.Connection == nil || h.groupService == nil || h.gameData == nil || h.gameData.GetMap(secretPassageMapID) == nil {
		return false, nil
	}
	if ctx.Connection.IsInBattle() {
		_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể dùng Sách Dịch Chuyển Mật Đạo khi đang chiến đấu.")
		return false, nil
	}
	inGroup, err := h.groupService.IsInGroup(ctx.Context, characterID)
	if err != nil {
		return false, err
	}
	if inGroup {
		_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể dùng Sách Dịch Chuyển Mật Đạo khi đang trong nhóm.")
		return false, nil
	}
	return true, nil
}
