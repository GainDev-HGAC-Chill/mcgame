package postgres

import (
	"errors"
	"testing"

	appactivity "mcgame-server/internal/application/activity"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapSendCombineCommandError(t *testing.T) {
	tests := []struct {
		message string
		want    error
	}{
		{message: "SEND_COMBINE_EXPIRED", want: appactivity.ErrSendCombineExpired},
		{message: "SEND_COMBINE_LIMIT_REACHED", want: appactivity.ErrSendCombineLimitReached},
		{message: "INSUFFICIENT_FUNDS", want: pkgerrors.ErrInsufficientFunds},
		{message: "INVENTORY_FULL", want: pkgerrors.ErrInventoryFull},
		{message: "INVALID_SEND_COMBINE_AWARD", want: pkgerrors.ErrInvalidInput},
	}
	for _, test := range tests {
		got := mapSendCombineCommandError(&pgconn.PgError{Message: test.message})
		if !errors.Is(got, test.want) {
			t.Fatalf("message %s mapped to %v, want %v", test.message, got, test.want)
		}
	}
}
