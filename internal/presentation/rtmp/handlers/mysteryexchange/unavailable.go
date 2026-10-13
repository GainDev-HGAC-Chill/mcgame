package mysteryexchange

import (
	"go.uber.org/zap"
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) ExchangeUnavailable(ctx *rtmp.RPCContext, _ []interface{}) (interface{}, error) {
	if ctx != nil && ctx.Connection != nil {
		if err := ctx.Connection.SendCallback("a", "Lò luyện thần bí chưa có cơ chế quy đổi an toàn; chưa trừ vật phẩm hoặc điểm."); err != nil {
			h.logger.Warn("Failed to report unavailable furnace exchange", zap.Error(err))
		}
	}
	return false, nil
}
