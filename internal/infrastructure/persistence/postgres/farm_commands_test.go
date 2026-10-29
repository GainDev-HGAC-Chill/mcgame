package postgres

import (
	"errors"
	"testing"

	appfarm "mcgame-server/internal/application/farm"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapFarmCommandError(t *testing.T) {
	tests := []struct {
		message string
		want    error
	}{
		{message: "FARM_PLOT_OCCUPIED", want: appfarm.ErrPlotOccupied},
		{message: "FARM_ANOTHER_PLOT_ACTIVE", want: appfarm.ErrAnotherPlotActive},
		{message: "FARM_PLOT_NOT_FOUND", want: appfarm.ErrPlotNotFound},
		{message: "FARM_CROP_NOT_READY", want: appfarm.ErrCropNotReady},
		{message: "FARM_SEED_NOT_FOUND", want: appfarm.ErrSeedNotFound},
		{message: "FARM_PLOT_ALREADY_SHARED", want: appfarm.ErrPlotAlreadyShared},
		{message: "FARM_PLOT_SHARED_OUT", want: appfarm.ErrPlotSharedOut},
		{message: "FARM_VIGOR_NOT_ENOUGH", want: pkgerrors.ErrInsufficientFunds},
		{message: "INVENTORY_FULL", want: pkgerrors.ErrInventoryFull},
	}
	for _, test := range tests {
		got := mapFarmCommandError(&pgconn.PgError{Message: test.message})
		if !errors.Is(got, test.want) {
			t.Fatalf("mapFarmCommandError(%q) = %v; want %v", test.message, got, test.want)
		}
	}
}
