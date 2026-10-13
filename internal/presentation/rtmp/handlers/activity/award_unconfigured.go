// Unconfigured event reads return supported empty states, never synthetic rewards or schedules.
package activity

import (
	"math"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) SerchForGift(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if _, _, err := h.characterID(ctx, "serchForGift"); err != nil {
		return nil, err
	}
	if len(args) != 0 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	return false, nil
}

func (h *Handler) GetConsume(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if _, _, err := h.characterID(ctx, "getConsume"); err != nil {
		return nil, err
	}
	return map[string]interface{}{}, nil
}

func (h *Handler) GetActAwardInfo(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if _, _, err := h.characterID(ctx, "getActAwardInfo"); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	var kind float64
	switch value := args[0].(type) {
	case float64:
		kind = value
	case int:
		kind = float64(value)
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}
	if math.IsNaN(kind) || math.IsInf(kind, 0) || math.Trunc(kind) != kind || kind < 0 || kind > 13 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	h.sendActivityCallback(ctx, "onSystemSay", "Sự kiện server mới chưa có cấu hình lịch và phần thưởng trên server local.")
	return nil, nil
}
