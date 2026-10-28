// Guild game-data topology characterization.
package guild

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

type guildCatalogFile[T any] struct {
	RecordCount int `json:"recordCount"`
	Records     []T `json:"records"`
}

type guildBuildingRecord struct {
	ID          string `json:"id"`
	CodeName    string `json:"codeName"`
	Level       string `json:"level"`
	Type        string `json:"type"`
	PreBuilding string `json:"preBuilding"`
	NumLimit    string `json:"numLimit"`
	MoneyCost   string `json:"moneyCost"`
	ExpCost     string `json:"expCost"`
	GenMCost    string `json:"genMCost"`
	RareMCost   string `json:"rareMCost"`
	SPMCost     string `json:"spMCost"`
	GoldCost    string `json:"goldCost"`
	Prop1       string `json:"prop1"`
	Prop1Value  string `json:"prop1Value"`
	Prop2       string `json:"prop2"`
	Prop2Value  string `json:"prop2Value"`
}

type guildExtendPositionRecord struct {
	Mid string `json:"mid"`
}

type guildSkillRecord struct {
	ID       string `json:"id"`
	CodeName string `json:"codeName"`
	Level    string `json:"level"`
	UseEnv   string `json:"useEnv"`
}

func readGuildCatalog[T any](t *testing.T, name string) guildCatalogFile[T] {
	t.Helper()
	raw, err := os.ReadFile("../../../../../docs/database/game_data/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var catalog guildCatalogFile[T]
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.RecordCount != len(catalog.Records) {
		t.Fatalf("%s recordCount = %d, decoded = %d", name, catalog.RecordCount, len(catalog.Records))
	}
	return catalog
}

func TestGuildCatalogTopology(t *testing.T) {
	buildings := readGuildCatalog[guildBuildingRecord](t, "TBL_BUILDING.json")
	if len(buildings.Records) != 18 {
		t.Fatalf("building count = %d, want 18", len(buildings.Records))
	}
	expectedBuildings := map[string]string{
		"2":  "BUILD100001|1||1|0|0|0|0|0|0|4:1|1:20",
		"3":  "BUILD100002|1|2|2|60000|192000|300|0|100|0|1:20|0:0",
		"4":  "BUILD100004|1|6|1|630000|1008000|1500|800|1000|0|0:0|0:0",
		"5":  "BUILD100003|1|2|1|0|0|0|0|0|0|5:1|0:0",
		"6":  "BUILD100001|2|2|3|5|1|1260000|2016000|3000|1600|2000|0|4:2|1:20",
		"7":  "BUILD100001|3|6|8|5|4|1|2400000|3840000|6000|3000|3200|0|4:3|1:20",
		"8":  "BUILD100002|2|6|3|2|126000|403200|300|160|200|0|1:30|0:0",
		"9":  "BUILD100002|3|7|8|2|240000|768000|600|300|320|0|1:40|0:0",
		"11": "BUILD100001|4|7|9|5|4|1|4275000|6840000|11000|5000|4600|0|4:4|1:20",
		"12": "BUILD100001|5|11|13|15|4|1|7200000|11520000|18000|9000|6200|0|4:5|1:20",
		"13": "BUILD100002|4|11|9|2|427500|1368000|1100|500|460|0|1:50|0:0",
		"14": "BUILD100002|5|12|13|2|720000|2304000|1800|900|620|0|1:60|0:0",
		"15": "BUILD100003|2|11|5|1|1282500|2052000|3300|1500|1380|0|5:2|0:0",
		"16": "BUILD100001|6|12|14|15|4|1|11550000|18480000|27000|14000|8000|0|4:6|1:20",
		"17": "BUILD100002|6|16|14|2|1155000|3696000|2700|1500|800|0|1:70|0:0",
		"18": "BUILD100004|2|4|16|1|5800000|9240000|13500|7200|4000|0|0:0|0:0",
	}
	playable := 0
	for _, building := range buildings.Records {
		if building.Type != "1" || building.Level == "0" {
			continue
		}
		playable++
		got := building.CodeName + "|" + building.Level + "|" + building.PreBuilding + "|" + building.NumLimit + "|" + building.MoneyCost + "|" + building.ExpCost + "|" + building.GenMCost + "|" + building.RareMCost + "|" + building.SPMCost + "|" + building.GoldCost + "|" + building.Prop1 + ":" + building.Prop1Value + "|" + building.Prop2 + ":" + building.Prop2Value
		if want, ok := expectedBuildings[building.ID]; !ok || got != want {
			t.Fatalf("building %s contract = %q, want %q", building.ID, got, want)
		}
	}
	if playable != 16 || len(expectedBuildings) != playable {
		t.Fatalf("playable building count = %d, expected contracts = %d, want 16", playable, len(expectedBuildings))
	}

	positions := readGuildCatalog[guildExtendPositionRecord](t, "TBL_EXTEND_POSITION.json")
	map49 := 0
	for _, position := range positions.Records {
		if position.Mid == "49" {
			map49++
		}
	}
	if map49 != 11 {
		t.Fatalf("map 49 extension count = %d, want 11", map49)
	}

	skills := readGuildCatalog[guildSkillRecord](t, "TBL_SKILL.json")
	families := map[string]map[int]guildSkillRecord{}
	for _, skill := range skills.Records {
		if skill.UseEnv != "4" {
			continue
		}
		level, err := strconv.Atoi(skill.Level)
		if err != nil {
			t.Fatalf("skill %s has invalid level %q", skill.ID, skill.Level)
		}
		family := skill.CodeName
		if level >= 7 {
			if len(family) < 2 || family[len(family)-2:] != "NO" {
				t.Fatalf("guild skill %s level %d codeName = %q, want NO boundary", skill.ID, level, family)
			}
			family = family[:len(family)-2]
		}
		if families[family] == nil {
			families[family] = map[int]guildSkillRecord{}
		}
		families[family][level] = skill
	}
	if len(families) != 13 {
		t.Fatalf("guild skill family count = %d, want 13", len(families))
	}
	total := 0
	for family, levels := range families {
		total += len(levels)
		if len(levels) != 10 {
			t.Fatalf("guild skill family %q level count = %d, want 10", family, len(levels))
		}
		if levels[6].CodeName != family || levels[7].CodeName != family+"NO" {
			t.Fatalf("guild skill family %q level 6/7 boundary = %q/%q", family, levels[6].CodeName, levels[7].CodeName)
		}
	}
	if total != 130 {
		t.Fatalf("guild skill count = %d, want 130", total)
	}
}
