package monsterheart

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNewStateBoxLevelsResolveClientTemplates(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/database/game_data/TBL_CREATUREH_CONTAIN.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]bool)
	for _, record := range table.Records {
		ids[record.ID] = true
	}
	state := NewState()
	for box := 1; box <= BoxCount; box++ {
		level := state.HeartBoxLev[box]
		if level != MinBoxLevel || !ids[intKey(box*10+level)] {
			t.Fatalf("box %d level %d has no baseline client contain template", box, level)
		}
	}
}

func TestDecodeStateBoxLevels(t *testing.T) {
	for _, level := range []interface{}{nil, -1, 0, 1, float64(3), 6, 7, "invalid"} {
		state := DecodeState(map[string]interface{}{
			"data": map[string]interface{}{
				"heartBoxLev": map[string]interface{}{"1": level},
			},
		})
		want := anyInt(level)
		if want < MinBoxLevel || want > MaxBoxLevel {
			want = MinBoxLevel
		}
		if state.HeartBoxLev[1] != want {
			t.Fatalf("level %#v decoded as %d, want %d", level, state.HeartBoxLev[1], want)
		}
		if state.HeartBoxLev[2] != MinBoxLevel {
			t.Fatal("missing levels must resolve to baseline level")
		}
	}
}
