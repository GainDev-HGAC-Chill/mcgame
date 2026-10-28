// Guild-loop rewards use the operator-approved Vua Pháp Thuật NEO calibrated profile.
package quest

import "fmt"

const (
	guildLoopMinRewardLevel   = 30
	guildLoopMaxRewardLevel   = 160
	guildLoopReferenceLevel   = 90
	guildLoopReferenceContrib = int64(2173)
)

var guildLoopRoundMultipliers = [...]int64{6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

type GuildLoopReward struct {
	Money        int64
	MoneyBind    int64
	Contribution int64
	GuildExp     int64
}

func CalculateGuildLoopReward(level, round int) (GuildLoopReward, error) {
	if level < guildLoopMinRewardLevel || level > guildLoopMaxRewardLevel {
		return GuildLoopReward{}, fmt.Errorf("guild loop reward level out of range: %d", level)
	}
	if round < 1 || round > 15 {
		return GuildLoopReward{}, fmt.Errorf("guild loop reward round out of range: %d", round)
	}

	multiplierIndex := round - 1
	if multiplierIndex >= len(guildLoopRoundMultipliers) {
		multiplierIndex = len(guildLoopRoundMultipliers) - 1
	}
	multiplier := guildLoopRoundMultipliers[multiplierIndex]
	levelSquared := int64(level * level)
	referenceSquared := int64(guildLoopReferenceLevel * guildLoopReferenceLevel)
	moneyBase := int64(48) * levelSquared
	contributionBase := roundPositive(guildLoopReferenceContrib*levelSquared, referenceSquared)
	money := roundPositive(moneyBase*multiplier, 10)
	contribution := roundPositive(contributionBase*multiplier, 10)

	return GuildLoopReward{
		Money:        money,
		MoneyBind:    money,
		Contribution: contribution,
		GuildExp:     contribution * 2,
	}, nil
}

func roundPositive(numerator, denominator int64) int64 {
	return (numerator + denominator/2) / denominator
}
