package quest

import "testing"

func TestCalculateGuildLoopRewardMatchesNEOLevel90Capture(t *testing.T) {
	tests := []struct {
		round        int
		money        int64
		contribution int64
		guildExp     int64
	}{
		{round: 1, money: 233280, contribution: 1304, guildExp: 2608},
		{round: 2, money: 272160, contribution: 1521, guildExp: 3042},
	}

	for _, test := range tests {
		reward, err := CalculateGuildLoopReward(90, test.round)
		if err != nil {
			t.Fatalf("round %d: %v", test.round, err)
		}
		if reward.Money != test.money || reward.MoneyBind != test.money || reward.Contribution != test.contribution || reward.GuildExp != test.guildExp {
			t.Fatalf("round %d reward = %+v, want money=%d contribution=%d guildExp=%d", test.round, reward, test.money, test.contribution, test.guildExp)
		}
	}
}

func TestCalculateGuildLoopRewardBoundariesAndRoundCap(t *testing.T) {
	tests := []struct {
		level int
		round int
	}{
		{level: 30, round: 1},
		{level: 30, round: 15},
		{level: 160, round: 1},
		{level: 160, round: 10},
		{level: 160, round: 15},
	}

	for _, test := range tests {
		reward, err := CalculateGuildLoopReward(test.level, test.round)
		if err != nil {
			t.Fatalf("level %d round %d: %v", test.level, test.round, err)
		}
		if reward.Money <= 0 || reward.MoneyBind != reward.Money || reward.Contribution <= 0 || reward.GuildExp != reward.Contribution*2 {
			t.Fatalf("level %d round %d invalid reward: %+v", test.level, test.round, reward)
		}
	}

	round10, err := CalculateGuildLoopReward(160, 10)
	if err != nil {
		t.Fatal(err)
	}
	round15, err := CalculateGuildLoopReward(160, 15)
	if err != nil {
		t.Fatal(err)
	}
	if round15 != round10 {
		t.Fatalf("round 15 reward = %+v, want capped round 10 reward %+v", round15, round10)
	}
}

func TestCalculateGuildLoopRewardRejectsUncalibratedInputs(t *testing.T) {
	for _, input := range []struct {
		level int
		round int
	}{{29, 1}, {161, 1}, {90, 0}, {90, 16}} {
		if _, err := CalculateGuildLoopReward(input.level, input.round); err == nil {
			t.Fatalf("level %d round %d: expected error", input.level, input.round)
		}
	}
}
