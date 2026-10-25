// Open-sourced by BaoLT

package achievement

import (
	"context"
	"encoding/json"
	"testing"

	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestAchievementPointsStateReaderCountsUniqueEnabledCompletions(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	rows := []models.AchievementTemplate{
		{ID: 1, Award: 600, Enable: 1},
		{ID: 2, Award: 500, Enable: 1},
		{ID: 3, Award: 9999, Enable: 0},
	}
	for _, row := range rows {
		loadAchievementModel(t, manager, row)
	}
	repo := &achievementTestRepo{rows: []*domainachievement.Progress{
		{AchievementID: 1, IsCompleted: true},
		{AchievementID: 1, IsCompleted: true},
		{AchievementID: 2, IsCompleted: true},
		{AchievementID: 3, IsCompleted: true},
		{AchievementID: 4, IsCompleted: false},
	}}
	proofs, err := NewAchievementPointsStateReader(repo, manager).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 5 || proofs[0].Progress != 1100 || !proofs[0].Done || proofs[1].Done {
		t.Fatalf("proofs = %#v", proofs)
	}
}

func loadAchievementModel(t *testing.T, manager *gamedata.Manager, model models.AchievementTemplate) {
	t.Helper()
	if err := manager.GetCache().LoadTable(models.TableAchievement, []json.RawMessage{mustAchievementJSON(t, model)}); err != nil {
		t.Fatalf("load achievement model: %v", err)
	}
}

var _ AchievementPointsRepository = (*achievementTestRepo)(nil)
