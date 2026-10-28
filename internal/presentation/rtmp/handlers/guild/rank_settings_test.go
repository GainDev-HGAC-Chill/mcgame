package guild

import (
	"errors"
	"strconv"
	"testing"

	appguild "mcgame-server/internal/application/guild"
	pkgerrors "mcgame-server/pkg/errors"
)

func TestParseGuildRankSettingsRequiresSixTypedFlags(t *testing.T) {
	values := map[string]interface{}{}
	for _, setting := range appguild.DefaultRankSettings() {
		suffix := strconv.Itoa(setting.Rank)
		values["name"+suffix] = setting.Name
		if setting.Rank == 1 {
			continue
		}
		values["canAdd"+suffix] = float64(0)
		values["canQuest"+suffix] = float64(1)
		values["canSlot"+suffix] = float64(0)
		values["canInfo"+suffix] = float64(1)
		values["canDel"+suffix] = float64(0)
		values["canDuty"+suffix] = float64(1)
	}
	settings, err := parseGuildRankSettings(values)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) != 6 || !settings[5].CanQuest || !settings[5].CanInfo || !settings[5].CanDuty {
		t.Fatalf("decoded settings = %#v", settings)
	}

	values["canSlot6"] = float64(2)
	if _, err := parseGuildRankSettings(values); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("invalid boolean error = %v", err)
	}
	values["canSlot6"] = float64(0.5)
	if _, err := parseGuildRankSettings(values); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("fractional boolean error = %v", err)
	}
	delete(values, "canSlot6")
	if _, err := parseGuildRankSettings(values); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("missing boolean error = %v", err)
	}
}
