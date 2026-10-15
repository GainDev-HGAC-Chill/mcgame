package main

import "testing"

func TestLevelBand(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{1, 0}, {29, 0}, {30, 1}, {54, 1}, {55, 2}, {89, 2},
		{90, 3}, {119, 3}, {120, 4}, {149, 4}, {150, 5}, {170, 5},
	}
	for _, test := range tests {
		if got := levelBand(test.level); got != test.want {
			t.Fatalf("levelBand(%d) = %d, want %d", test.level, got, test.want)
		}
	}
}

func TestRewardProfiles(t *testing.T) {
	tests := []struct {
		boss bossRow
		want string
	}{
		{bossRow{NID: 706, Kind: "ground", Tier: "normal"}, "ground"},
		{bossRow{NID: 1638, Kind: "flying", Tier: "normal"}, "flying"},
		{bossRow{NID: 2221, Kind: "daily_only", Tier: "normal"}, "flying"},
		{bossRow{NID: 2204, Kind: "daily_only", Tier: "normal"}, "ground"},
		{bossRow{NID: 2262, Kind: "daily_only", Tier: "mythic"}, "mythic"},
	}
	for _, test := range tests {
		if got := rewardProfile(test.boss).name; got != test.want {
			t.Fatalf("rewardProfile(%+v) = %q, want %q", test.boss, got, test.want)
		}
	}
}

func TestBuildLootCoversEveryBoss(t *testing.T) {
	bosses := []bossRow{
		{NID: 706, Level: 5, Kind: "ground", Tier: "normal"},
		{NID: 1638, Level: 65, Kind: "flying", Tier: "normal"},
		{NID: 2262, Level: 95, Kind: "daily_only", Tier: "mythic"},
	}
	loot := buildLoot(bosses)
	covered := make(map[int]int)
	for _, row := range loot {
		covered[row.BossNID]++
		if row.Rate < 1 || row.Rate > 50000 {
			t.Fatalf("invalid rate %+v", row)
		}
		if row.QtyMin < 1 || row.QtyMax < row.QtyMin {
			t.Fatalf("invalid quantity %+v", row)
		}
	}
	for _, boss := range bosses {
		if covered[boss.NID] == 0 {
			t.Fatalf("boss %d has no loot", boss.NID)
		}
	}
}
