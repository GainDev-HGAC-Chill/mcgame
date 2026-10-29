// Wedding booking and hall-entry RTMP handlers.
package marriage

import (
	"errors"
	"time"

	appmarriage "mcgame-server/internal/application/marriage"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func (h *Handler) BookWeddingHall(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	payload, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	line, lineErr := parseIntArg(payload["line"])
	hour, hourErr := parseIntArg(payload["hour"])
	minute, minuteErr := parseIntArg(payload["minute"])
	flag, flagErr := parseIntArg(payload["flag"])
	if lineErr != nil || hourErr != nil || minuteErr != nil || flagErr != nil {
		return nil, pkgerrors.ErrInvalidArgs
	}
	bookings, err := h.service.BookWeddingHall(ctx.Context, characterID, line, hour, minute, flag, time.Now())
	if err != nil {
		h.sendWeddingError(ctx, err)
		return rtmp.ErrorToResponse(err), nil
	}
	if err := ctx.Connection.SendCallback("onInitBookList", bookings); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) CancelWeddingHall(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	bookings, err := h.service.CancelWeddingHall(ctx.Context, characterID, time.Now())
	if err != nil {
		h.sendWeddingError(ctx, err)
		return rtmp.ErrorToResponse(err), nil
	}
	if err := ctx.Connection.SendCallback("onInitBookList", bookings); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) EnterWeddingHall(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	groomID, err := parseInt64Arg(args[0])
	if err != nil || groomID <= 0 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	entry, err := h.service.EnterWeddingHall(ctx.Context, characterID, groomID, ctx.Connection.GetChannelID(), time.Now())
	if err != nil {
		h.sendWeddingError(ctx, err)
		return rtmp.ErrorToResponse(err), nil
	}
	if h.transport == nil || !h.transport.TransportPlayerAlone(characterID, entry.MapID, 800, 1400) {
		h.logger.Warn("EnterWeddingHall: transport unavailable", zap.Int64("character_id", characterID), zap.Int("map_id", entry.MapID))
		ctx.Connection.SendCallback("onSystemSay", "Không thể vào lễ đường lúc này.")
		return rtmp.ErrorToResponse(pkgerrors.ErrSystemError), nil
	}
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) sendWeddingError(ctx *rtmp.RPCContext, err error) {
	message := "Không thể thực hiện thao tác hôn lễ."
	switch {
	case errors.Is(err, appmarriage.ErrNotMarried):
		message = "Bạn chưa kết hôn."
	case errors.Is(err, appmarriage.ErrWeddingAlreadyBooked):
		message = "Bạn đã đặt lễ đường."
	case errors.Is(err, appmarriage.ErrWeddingSlotTaken):
		message = "Khung giờ này đã được đặt."
	case errors.Is(err, appmarriage.ErrWeddingNotBooked):
		message = "Không tìm thấy lịch đặt lễ đường."
	case errors.Is(err, appmarriage.ErrWeddingNotOpen):
		message = "Lễ đường chưa đến thời gian mở cửa."
	case errors.Is(err, appmarriage.ErrWeddingWrongLine):
		message = "Hôn lễ được tổ chức ở kênh khác."
	case errors.Is(err, appmarriage.ErrWeddingInvitation):
		message = "Bạn cần thiệp mời phù hợp để vào lễ đường."
	case errors.Is(err, pkgerrors.ErrInsufficientFunds):
		message = "Bạn không đủ bạc để đặt lễ đường."
	case errors.Is(err, pkgerrors.ErrInvalidArgs), errors.Is(err, pkgerrors.ErrInvalidInput):
		message = "Thời gian hoặc thông tin đặt lễ đường không hợp lệ."
	}
	_ = ctx.Connection.SendCallback("onSystemSay", message)
}
