// Open-sourced by BaoLT

// getDecoSuitProp handler — returns decoration suit set-bonus properties.
// Args: [suitId:int, linkSuitId:int, linkId:int]
// Reply: {actName, suitIdObj, suitProp, actArr, actFlag, linkProp, actLinkSuitFlag, actLinkSuitName}
// actArr and actFlag are stubbed empty/false until character_decorations is
// wired into the character load path (player activation state not tracked yet).
package dress

import (
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) GetDecoSuitProp(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	charID, validCharacter := h.characterID(ctx)
	if !validCharacter || h.decoHoleService == nil {
		return nil, nil
	}
	if len(args) < 3 {
		return nil, nil
	}
	suitID, ok := toInt64(args[0])
	if !ok || suitID <= 0 {
		return nil, nil
	}
	linkSuitID, _ := toInt64(args[1])
	linkID, _ := toInt64(args[2])

	result, err := h.decoHoleService.GetDecoSuitProp(ctx.Context, charID, int(suitID), int(linkSuitID), int(linkID))
	if err != nil || result == nil {
		if err != nil {
			h.logger.Error("getDecoSuitProp: service error", zap.Int64("character_id", charID), zap.Error(err))
		}
		return nil, nil
	}

	reply := map[string]interface{}{
		"actName":         result.ActName,
		"suitIdObj":       result.SuitIdObj,
		"suitProp":        result.SuitProp,
		"actArr":          result.ActArr,
		"actFlag":         result.ActFlag,
		"linkProp":        result.LinkProp,
		"actLinkSuitFlag": result.ActLinkSuitFlag,
		"actLinkSuitName": result.ActLinkSuitName,
	}
	if result.LinkFlag {
		reply["linkFlag"] = result.LinkFlag
	}
	return reply, nil
}
