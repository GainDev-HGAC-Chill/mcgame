// Open-sourced by BaoLT

package soul

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
)

func TestPetSoulVinhSinhLocalizationRows(t *testing.T) {
	content, err := os.ReadFile("../../../docs/database/game_data/TBL_PET_SOUL.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Records []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Desc    string `json:"desc"`
			PropVal string `json:"propVal"`
		} `json:"records"`
	}
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, row := range catalog.Records {
		id, err := strconv.Atoi(row.ID)
		if err != nil || id < 751 || id > 760 {
			continue
		}
		seen++
		if row.Name != "Vĩnh Sinh" || row.Desc != fmt.Sprintf("Tăng Miễn Tử %s", row.PropVal) {
			t.Fatalf("row %s localization=%q/%q", row.ID, row.Name, row.Desc)
		}
	}
	if seen != 10 {
		t.Fatalf("localized rows=%d want 10", seen)
	}
}
