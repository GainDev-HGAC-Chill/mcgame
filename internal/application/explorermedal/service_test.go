package explorermedal

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type medalStateRepo struct {
	mu      sync.Mutex
	state   *domainfeature.CharacterFeatureState
	saveErr error
}

func (r *medalStateRepo) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == nil {
		return nil, nil
	}
	copy := *r.state
	copy.State = cloneMedalMap(r.state.State)
	return []*domainfeature.CharacterFeatureState{&copy}, nil
}

func (r *medalStateRepo) UpsertCharacterFeatureState(_ context.Context, state *domainfeature.CharacterFeatureState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	copy := *state
	copy.State = cloneMedalMap(state.State)
	r.state = &copy
	return nil
}

type medalCharacterRepo struct {
	mu   sync.Mutex
	char *domainchar.Character
}

func (r *medalCharacterRepo) FindByID(context.Context, int64) (*domainchar.Character, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *r.char
	return &copy, nil
}

func (r *medalCharacterRepo) Update(_ context.Context, char *domainchar.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *char
	r.char = &copy
	return nil
}

func TestPanelResponseSeparatesWalletAndProgress(t *testing.T) {
	service, stateRepo, _ := newMedalService(t, 250, 0)
	stateRepo.state = medalState(1, 35)
	panel, err := service.GetPanelResponse(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if panel[0] != 1 || panel[2] != 250 || panel[3] != 35 {
		t.Fatalf("panel=%v", panel)
	}
}

func TestPointUpgradeDepositsAvailableWalletAndLevelsAtThreshold(t *testing.T) {
	service, _, chars := newMedalService(t, 40, 0)
	partial, err := service.LevelUp(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Level != 0 || partial.Progress != 40 || partial.ExplorerPnt != 0 || partial.WalletSpent != 40 {
		t.Fatalf("partial=%+v", partial)
	}
	chars.mu.Lock()
	chars.char.ShowTimePoint = 80
	chars.mu.Unlock()
	complete, err := service.LevelUp(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if complete.Level != 1 || complete.Progress != 0 || complete.ExplorerPnt != 20 || complete.WalletSpent != 60 {
		t.Fatalf("complete=%+v", complete)
	}
}

func TestGoldUpgradeUsesFixedPricesAndCarriesAcrossLevels(t *testing.T) {
	service, _, _ := newMedalService(t, 0, 1000)
	first, err := service.LevelUpByGold(context.Background(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.Level != 0 || first.Progress != 50 || first.Gold != 950 || first.GoldSpent != 50 {
		t.Fatalf("first=%+v", first)
	}
	bulk, err := service.LevelUpByGold(context.Background(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if bulk.Level != 2 || bulk.Progress != 250 || bulk.Gold != 450 || bulk.GoldSpent != 500 {
		t.Fatalf("bulk=%+v", bulk)
	}
}

func TestFeatureSaveFailureCompensatesWallets(t *testing.T) {
	service, states, chars := newMedalService(t, 100, 100)
	states.saveErr = errors.New("save failed")
	if _, err := service.LevelUp(context.Background(), 1); err == nil {
		t.Fatal("point upgrade unexpectedly succeeded")
	}
	if chars.char.ShowTimePoint != 100 {
		t.Fatalf("explorer points=%d", chars.char.ShowTimePoint)
	}
	if _, err := service.LevelUpByGold(context.Background(), 1, 0); err == nil {
		t.Fatal("gold upgrade unexpectedly succeeded")
	}
	if chars.char.Gold != 100 {
		t.Fatalf("gold=%d", chars.char.Gold)
	}
}

func TestConcurrentGoldUpgradeSerializesPerCharacter(t *testing.T) {
	service, _, chars := newMedalService(t, 0, 100)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.LevelUpByGold(context.Background(), 1, 0)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	level, progress, err := service.GetState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if level != 1 || progress != 0 || chars.char.Gold != 0 {
		t.Fatalf("state=%d/%d gold=%d", level, progress, chars.char.Gold)
	}
}

func newMedalService(t *testing.T, explorerPnt int, gold int64) (*Service, *medalStateRepo, *medalCharacterRepo) {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	rows := []json.RawMessage{
		json.RawMessage(`{"id":1,"cost":100,"cost_gold":100,"prop_type1":1,"prop_num1":10}`),
		json.RawMessage(`{"id":2,"cost":200,"cost_gold":200,"prop_type1":1,"prop_num1":20}`),
		json.RawMessage(`{"id":3,"cost":300,"cost_gold":300,"prop_type1":1,"prop_num1":30}`),
	}
	if err := manager.GetCache().LoadTable(models.TableExplorerMedal, rows); err != nil {
		t.Fatal(err)
	}
	states := &medalStateRepo{}
	chars := &medalCharacterRepo{char: &domainchar.Character{ID: 1, ShowTimePoint: explorerPnt, Gold: gold}}
	return NewService(states, chars, manager), states, chars
}

func medalState(level, progress int) *domainfeature.CharacterFeatureState {
	return &domainfeature.CharacterFeatureState{CharacterID: 1, FeatureKey: domainfeature.FeatureExplorerMedal, State: map[string]interface{}{"info": strconv.Itoa(level) + "|" + strconv.Itoa(progress)}}
}

func cloneMedalMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
