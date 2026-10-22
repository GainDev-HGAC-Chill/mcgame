package monsterheart

import (
	"context"
	"testing"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type panelReadRepository struct {
	domainfeature.Repository
	state  *domainfeature.CharacterFeatureState
	writes int
}

type testCatalog struct {
	hearts map[int]*models.CreaturehHeartTemplate
	points map[int]*models.CreaturehPointTemplate
}

func (c testCatalog) GetCreatureHeart(id int) *models.CreaturehHeartTemplate {
	return c.hearts[id]
}

func (c testCatalog) GetCreatureHeartPoint(id int) *models.CreaturehPointTemplate {
	return c.points[id]
}

func (r *panelReadRepository) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	return []*domainfeature.CharacterFeatureState{r.state}, nil
}

func (r *panelReadRepository) UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error {
	r.writes++
	return nil
}

type mutationRepository struct {
	domainfeature.Repository
	state *domainfeature.CharacterFeatureState
}

func (r *mutationRepository) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	return []*domainfeature.CharacterFeatureState{r.state}, nil
}

func (r *mutationRepository) UpsertCharacterFeatureState(_ context.Context, state *domainfeature.CharacterFeatureState) error {
	r.state = state
	return nil
}

func TestLoadNormalizesLegacyBoxLevelWithoutWritingOrLosingState(t *testing.T) {
	levels := map[string]interface{}{"1": float64(0), "2": float64(4)}
	repo := &panelReadRepository{state: &domainfeature.CharacterFeatureState{
		CharacterID: 5,
		FeatureKey:  domainfeature.FeatureMonsterHeart,
		State: map[string]interface{}{
			"bag": map[string]interface{}{
				"ren": map[string]interface{}{
					"5001": map[string]interface{}{"itemId": float64(5001), "n": float64(2)},
				},
			},
			"data": map[string]interface{}{
				"heartBoxLev": levels,
				"hadActHole":  map[string]interface{}{"2": map[string]interface{}{"1": float64(5001)}},
			},
		},
	}}
	state, err := NewService(repo, zap.NewNop()).Load(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if state.HeartBoxLev[1] != 1 || state.HeartBoxLev[2] != 4 {
		t.Fatalf("loaded levels = %#v", state.HeartBoxLev)
	}
	if state.BagItemCount(5001) != 2 || state.GetHoleItem(2, 1) != 5001 {
		t.Fatal("panel load lost persisted bag or slot state")
	}
	if repo.writes != 0 || levels["1"] != float64(0) {
		t.Fatal("panel load must not repair persisted state or mutate repository input")
	}
}

func TestSetSlotRequiresCatalogBackedOwnedEntryAndOpenHole(t *testing.T) {
	repo := &mutationRepository{state: &domainfeature.CharacterFeatureState{
		CharacterID: 8,
		FeatureKey:  domainfeature.FeatureMonsterHeart,
		State: map[string]interface{}{
			"bag": map[string]interface{}{
				"ren": map[string]interface{}{
					"9001": map[string]interface{}{"id": 9001, "itemId": 101, "n": 1},
				},
			},
			"data": map[string]interface{}{
				"hadActHole": map[string]interface{}{"1": map[string]interface{}{"1": -1, "2": -1, "3": 0}},
			},
		},
	}}
	service := NewService(repo, zap.NewNop())
	service.SetCatalog(testCatalog{
		hearts: map[int]*models.CreaturehHeartTemplate{101: {ID: 101, Type: 1}},
		points: map[int]*models.CreaturehPointTemplate{
			11: {ID: 11, Type: -1},
			12: {ID: 12, Type: -1},
			13: {ID: 13, Type: -1},
		},
	})

	state, err := service.SetSlot(context.Background(), 8, 9001, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if state.GetHoleItem(1, 1) != 101 {
		t.Fatalf("equipped id = %d, want template 101", state.GetHoleItem(1, 1))
	}
	if _, err := service.SetSlot(context.Background(), 8, 9001, 2, 1, 1); err != ErrHeartUnavailable {
		t.Fatalf("duplicate placement error = %v, want %v", err, ErrHeartUnavailable)
	}
	if _, err := service.SetSlot(context.Background(), 8, 9001, 3, 1, 1); err != ErrHoleClosed {
		t.Fatalf("closed-hole error = %v, want %v", err, ErrHoleClosed)
	}
	if _, err := service.SetSlot(context.Background(), 8, 9999, 2, 1, 1); err != ErrItemNotInBag {
		t.Fatalf("forged entry error = %v, want %v", err, ErrItemNotInBag)
	}
	if _, err := service.SetSlot(context.Background(), 8, 9001, 2, 1, 2); err != ErrHeartType {
		t.Fatalf("forged type error = %v, want %v", err, ErrHeartType)
	}
}

func TestAddHeartPersistsInCatalogCategoryAndStacks(t *testing.T) {
	repo := &mutationRepository{}
	service := NewService(repo, zap.NewNop())
	service.SetCatalog(testCatalog{hearts: map[int]*models.CreaturehHeartTemplate{
		101: {ID: 101, Type: 4},
	}})

	panel, err := service.AddHeart(context.Background(), 8, 101, 2)
	if err != nil {
		t.Fatal(err)
	}
	state := DecodeState(panel)
	entry, ok := state.Bag["ji"][101]
	if !ok || entry.ID != 101 || entry.ItemID != 101 || entry.N != 2 {
		t.Fatalf("first grant = %#v", state.Bag["ji"])
	}

	panel, err = service.AddHeart(context.Background(), 8, 101, 3)
	if err != nil {
		t.Fatal(err)
	}
	state = DecodeState(panel)
	if got := state.Bag["ji"][101].N; got != 5 {
		t.Fatalf("stack = %d, want 5", got)
	}
	if _, err := service.AddHeart(context.Background(), 8, 999, 1); err != ErrCatalogEntry {
		t.Fatalf("unknown heart error = %v, want %v", err, ErrCatalogEntry)
	}
}
