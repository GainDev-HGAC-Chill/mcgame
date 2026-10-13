// Fail closed until account debit and cached character credit share a safe transaction flow.
package exchange

import (
	"math"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/google/uuid"
)

func (h *Handler) ExchangeAwardToGoldBind(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if _, err := uuid.Parse(ctx.AccountID); err != nil || ctx.CharacterID == "" {
		return nil, pkgerrors.ErrUnauthorized
	}
	if len(args) != 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	amount, ok := args[0].(float64)
	if !ok || math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || math.Trunc(amount) != amount {
		return nil, pkgerrors.ErrInvalidArgs
	}
	if ctx.Connection != nil {
		if err := ctx.Connection.SendCallback("onSystemSay", "Đổi phiếu sang kim phiếu khóa chưa được triển khai giao dịch an toàn. Chưa trừ phiếu hoặc cộng tiền."); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
