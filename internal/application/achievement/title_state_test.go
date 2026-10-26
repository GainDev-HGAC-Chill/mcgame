// Open-sourced by BaoLT

package achievement

import (
	"context"
	"encoding/json"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type titleStateRepositoryFixture struct {
	titles        []int
	specialTitles []int
}

func (r titleStateRepositoryFixture) GetCharacterTitles(ctx context.Context, characterID int64) ([]int, error) {
	return r.titles, nil
}

func (r titleStateRepositoryFixture) GetCharacterSpecialTitles(ctx context.Context, characterID int64) ([]int, error) {
	return r.specialTitles, nil
}

func TestTitleStateReaderCountsDistinctPositiveTitles(t *testing.T) {
	repo := titleStateRepositoryFixture{
		titles:        []int{1, 2, 3, 4, 5, 0, 144, 325},
		specialTitles: []int{5, 6, 7, 8, 9, 10},
	}
	manager := gamedata.NewManager(nil, zap.NewNop())
	load := func(table string, value interface{}) {
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Fatalf("marshal %s: %v", table, marshalErr)
		}
		if loadErr := manager.GetCache().LoadTable(table, []json.RawMessage{raw}); loadErr != nil {
			t.Fatalf("load %s: %v", table, loadErr)
		}
	}
	for _, definition := range exactTitleRequirements {
		load(models.TableTitle, models.TitleTemplate{ID: int64(definition.titleID), B: float64(definition.buffID), N: definition.titleName, L: float64(definition.level)})
		load(models.TableBuff, models.BuffTemplate{ID: int64(definition.buffID), Name: definition.buffName})
	}
	proofs, err := NewTitleStateReader(repo, manager).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if len(proofs) != 35 || byID[84].Done || !byID[50].Done || !byID[51].Done || byID[52].Done || byID[52].Progress != 12 || byID[128].Done || byID[163].Done || !byID[383].Done || !byID[582].Done || byID[584].Done {
		t.Fatalf("proofs = %#v", proofs)
	}
}

var _ TitleStateRepository = titleStateRepositoryFixture{}
