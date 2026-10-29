package postgres

import (
	"errors"
	"testing"

	appgathering "mcgame-server/internal/application/gathering"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapGatheringCommandError(t *testing.T) {
	tests := []struct {
		message string
		want    error
	}{
		{message: "GATHERING_RESOURCE_UNAVAILABLE", want: appgathering.ErrResourceUnavailable},
		{message: "GATHERING_SKILL_LEVEL_NOT_ENOUGH", want: appgathering.ErrSkillLevelNotEnough},
		{message: "GATHERING_TOOL_NOT_EQUIPPED", want: appgathering.ErrToolNotEquipped},
		{message: "GATHERING_TOOL_BROKEN", want: appgathering.ErrToolBroken},
		{message: "GATHERING_VIGOR_NOT_ENOUGH", want: appgathering.ErrVigorNotEnough},
		{message: "INVENTORY_FULL", want: pkgerrors.ErrInventoryFull},
	}
	for _, test := range tests {
		got := mapGatheringCommandError(&pgconn.PgError{Message: test.message})
		if !errors.Is(got, test.want) {
			t.Fatalf("mapGatheringCommandError(%q) = %v; want %v", test.message, got, test.want)
		}
	}
}
