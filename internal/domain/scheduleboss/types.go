// Open-sourced by BaoLT

// Types and constants for the Schedule Boss system.
// BossConfig is seeded from data.data_tbl_schedule_boss.
// BossState is the runtime row in player.schedule_boss_state.
package scheduleboss

import "time"

type Kind string

const (
	Ground Kind = "ground"
	Flying Kind = "flying"
)

type Tier string

const (
	TierNormal  Tier = "normal"
	TierMythic  Tier = "mythic"
	TierSpecial Tier = "special"
)

type BossConfig struct {
	NID          int
	Name         string
	MapID        int
	Level        int
	Kind         Kind
	Tier         Tier
	ChannelScope string
	Description  string
	IsActive     bool
}

func (c BossConfig) Eligible() bool {
	return c.IsMapBoss() && c.Tier == TierNormal
}

func (c BossConfig) IsMapBoss() bool {
	return c.IsActive && (c.Kind == Ground || c.Kind == Flying)
}

func (c BossConfig) AllowsChannel(channelID int) bool {
	if channelID <= 0 {
		return false
	}
	if c.ChannelScope == "" || c.ChannelScope == "all" {
		return true
	}
	const prefix = "channel:"
	if len(c.ChannelScope) <= len(prefix) || c.ChannelScope[:len(prefix)] != prefix {
		return false
	}
	allowed := 0
	for _, ch := range c.ChannelScope[len(prefix):] {
		if ch < '0' || ch > '9' {
			return false
		}
		allowed = allowed*10 + int(ch-'0')
	}
	return allowed == channelID
}

type BossState struct {
	NID          int
	ChannelID    int
	IsAlive      bool
	LastKilledAt *time.Time
	LastKillerID *int64
	NextSpawnAt  time.Time
	SpawnCount   int
	UpdatedAt    time.Time
}
