package scheduleboss

import "testing"

func TestBossConfigEligibleExcludesDailyOnly(t *testing.T) {
	tests := []struct {
		name string
		cfg  BossConfig
		want bool
	}{
		{"ground normal", BossConfig{Kind: Ground, Tier: TierNormal, IsActive: true}, true},
		{"flying normal", BossConfig{Kind: Flying, Tier: TierNormal, IsActive: true}, true},
		{"daily normal", BossConfig{Kind: Kind("daily_only"), Tier: TierNormal, IsActive: true}, false},
		{"ground mythic", BossConfig{Kind: Ground, Tier: TierMythic, IsActive: true}, false},
		{"inactive ground", BossConfig{Kind: Ground, Tier: TierNormal}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.cfg.Eligible(); got != test.want {
				t.Fatalf("Eligible() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBossConfigAllowsChannel(t *testing.T) {
	tests := []struct {
		scope   string
		channel int
		want    bool
	}{
		{"all", 3, true},
		{"", 3, true},
		{"channel:1", 1, true},
		{"channel:1", 2, false},
		{"all", 0, true},
		{"channel:1", 0, false},
		{"channel:x", 1, false},
		{"all", -1, false},
	}
	for _, test := range tests {
		cfg := BossConfig{ChannelScope: test.scope}
		if got := cfg.AllowsChannel(test.channel); got != test.want {
			t.Fatalf("AllowsChannel(%q, %d) = %v, want %v", test.scope, test.channel, got, test.want)
		}
	}
}
