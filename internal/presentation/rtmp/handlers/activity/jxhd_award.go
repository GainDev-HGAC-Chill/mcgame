// Featured-activity preview RPC returns the client's explicit empty-preview shape.
// No preview metadata is configured here; this is read-only, not a reward claim.
package activity

import (
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func (h *Handler) GetJXHDAward(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if ctx == nil || ctx.CharacterID == "" {
		return nil, pkgerrors.ErrUnauthorized
	}
	if len(args) != 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	key, ok := args[0].(string)
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	switch key {
	case "ConsumeAward", "Wawagame", "PointAward", "ComboShop", "Juhuasuan", "Treasurebowl", "Happyline", "PointNotice":
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}
	h.logger.Debug("Featured activity reward preview unavailable", zap.String("activity", key), zap.String("character_id", ctx.CharacterID))
	return []interface{}{key, []interface{}{}}, nil
}
