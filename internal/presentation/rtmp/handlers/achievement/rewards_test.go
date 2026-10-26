// Open-sourced by BaoLT

package achievement

import (
	"testing"

	appachievement "mcgame-server/internal/application/achievement"
	domainachievement "mcgame-server/internal/domain/achievement"
)

func TestValidateRewardReceipt(t *testing.T) {
	tests := []struct {
		name          string
		achievementID int64
		receipt       domainachievement.RewardReceipt
		wantError     bool
	}{
		{name: "item", achievementID: 1008, receipt: domainachievement.RewardReceipt{ItemID: 10}},
		{name: "title", achievementID: 6066, receipt: domainachievement.RewardReceipt{TitleID: 274}},
		{name: "medal", achievementID: 1015, receipt: domainachievement.RewardReceipt{MedalSlot: 1, MedalCount: 1}},
		{name: "wrong title", achievementID: 6066, receipt: domainachievement.RewardReceipt{TitleID: 273}, wantError: true},
		{name: "missing item", achievementID: 1008, receipt: domainachievement.RewardReceipt{}, wantError: true},
		{name: "unknown reward", achievementID: 6067, receipt: domainachievement.RewardReceipt{}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := appachievement.ValidateRewardReceipt(test.achievementID, test.receipt)
			if (err != nil) != test.wantError {
				t.Fatalf("validateRewardReceipt() error = %v, wantError=%t", err, test.wantError)
			}
		})
	}
}
