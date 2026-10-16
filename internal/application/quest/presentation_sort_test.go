package quest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestEnrichQuestDTOIncludesTemplateSortColor(t *testing.T) {
	for _, color := range []float64{0, 1, 4} {
		for _, questType := range []float64{1, 9, 10} {
			t.Run(fmt.Sprintf("type%.0f/color%.0f", questType, color), func(t *testing.T) {
				logger := zap.NewNop()
				manager := gamedata.NewManager(nil, logger)
				template := models.QuestTemplate{ID: 3165, Type: questType, Color: color}
				if err := manager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, template)}); err != nil {
					t.Fatal(err)
				}
				svc := NewService(nil, logger)
				svc.SetGameDataManager(manager)
				progress := domainquest.NewQuestProgress(1, 3165, nil)
				dto := svc.EnrichQuestDTO(context.Background(), progress)
				data := dto["data"].(map[string]interface{})
				if got, ok := data["color"].(int); !ok || got != int(color) {
					t.Fatalf("sort1 must be the numeric template color, got %#v", data["color"])
				}
				if _, ok := dto["state"].(int); !ok {
					t.Fatalf("sort2 must be numeric state, got %#v", dto["state"])
				}
				if len(progress.Objectives) != 0 || dto["c"] != int(color) {
					t.Fatalf("presentation changed objectives or callboard color: %#v", dto)
				}
			})
		}
	}
}

func TestEnrichQuestDTOUsesCancellablePresentationTypeForNewbieQuest(t *testing.T) {
	logger := zap.NewNop()
	manager := gamedata.NewManager(nil, logger)
	template := models.QuestTemplate{ID: 3165, Type: 1, Color: 1}
	if err := manager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, template)}); err != nil {
		t.Fatal(err)
	}

	svc := NewService(nil, logger)
	svc.SetGameDataManager(manager)
	dto := svc.EnrichQuestDTO(context.Background(), domainquest.NewQuestProgress(1, 3165, nil))
	data := dto["data"].(map[string]interface{})
	if got := data["type"]; got != clientQuestTypeMain {
		t.Fatalf("client quest type = %#v, want %d so Flash enables cancel", got, clientQuestTypeMain)
	}
	if got := manager.GetQuest(3165); got == nil || int(got.Type) != 1 {
		t.Fatalf("canonical quest type changed: %#v", got)
	}
}
