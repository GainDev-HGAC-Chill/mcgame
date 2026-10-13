package bloodbag

import (
	"context"
	"errors"
	"sync"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
)

type testCharacterRepo struct {
	mu      sync.Mutex
	char    *domainchar.Character
	updates int
	err     error
}

func (r *testCharacterRepo) FindByID(context.Context, int64) (*domainchar.Character, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *r.char
	return &copy, nil
}

func (r *testCharacterRepo) Update(_ context.Context, char *domainchar.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	copy := *char
	r.char = &copy
	r.updates++
	return nil
}

type testFeatureRepo struct {
	mu    sync.Mutex
	state map[string]interface{}
	err   error
}

type atomicTestFeatureRepo struct {
	*testFeatureRepo
	commits int
	gold    int64
}

func (r *atomicTestFeatureRepo) CommitBloodBagPurchase(_ context.Context, _ int64, newGold int64, state map[string]interface{}) error {
	if r.err != nil {
		return r.err
	}
	r.commits++
	r.gold = newGold
	r.state = state
	return nil
}

func (r *testFeatureRepo) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := make(map[string]interface{}, len(r.state))
	for key, value := range r.state {
		copy[key] = value
	}
	return []*domainfeature.CharacterFeatureState{{CharacterID: 1, FeatureKey: domainfeature.FeatureBloodBag, State: copy}}, nil
}

func (r *testFeatureRepo) UpsertCharacterFeatureState(_ context.Context, state *domainfeature.CharacterFeatureState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.state = state.State
	return nil
}

func TestPurchaseValidatesCostCapacityAndPersists(t *testing.T) {
	chars := &testCharacterRepo{char: &domainchar.Character{ID: 1, Gold: 5}}
	features := &testFeatureRepo{state: map[string]interface{}{"characterHp": int64(100)}}
	service := NewService(chars, features)
	result, err := service.Purchase(context.Background(), 1, 1, 5001)
	if err != nil {
		t.Fatal(err)
	}
	if result.Cost != 2 || result.NewGold != 3 || result.State.CharacterHP != 5101 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.State.String() != "|5101|0|0|0" {
		t.Fatalf("serialized state = %q", result.State.String())
	}
	if _, err := service.Purchase(context.Background(), 1, 1, Capacity); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestPurchaseRejectsInsufficientGoldWithoutMutation(t *testing.T) {
	chars := &testCharacterRepo{char: &domainchar.Character{ID: 1, Gold: 1}}
	features := &testFeatureRepo{state: map[string]interface{}{}}
	service := NewService(chars, features)
	if _, err := service.Purchase(context.Background(), 1, 2, 5001); !errors.Is(err, ErrInsufficientGold) {
		t.Fatalf("error = %v", err)
	}
	if chars.char.Gold != 1 || len(features.state) != 0 {
		t.Fatalf("state mutated: gold=%d state=%v", chars.char.Gold, features.state)
	}
}

func TestPurchaseRollsBackGoldWhenStateWriteFails(t *testing.T) {
	chars := &testCharacterRepo{char: &domainchar.Character{ID: 1, Gold: 3}}
	features := &testFeatureRepo{state: map[string]interface{}{}, err: errors.New("write failed")}
	service := NewService(chars, features)
	if _, err := service.Purchase(context.Background(), 1, 3, 5000); err == nil {
		t.Fatal("expected persistence error")
	}
	if chars.char.Gold != 3 || chars.updates != 2 {
		t.Fatalf("rollback failed: gold=%d updates=%d", chars.char.Gold, chars.updates)
	}
}

func TestConcurrentPurchaseSerializesCapacityCheck(t *testing.T) {
	chars := &testCharacterRepo{char: &domainchar.Character{ID: 1, Gold: 10_000}}
	features := &testFeatureRepo{state: map[string]interface{}{"petMp": Capacity - 5000}}
	service := NewService(chars, features)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := service.Purchase(context.Background(), 1, 4, 5000)
			results <- err
		}()
	}
	var successes int
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 || StateFromMap(features.state).PetMP != Capacity {
		t.Fatalf("successes=%d state=%v", successes, features.state)
	}
}

func TestPurchaseUsesAtomicRepositoryWhenAvailable(t *testing.T) {
	chars := &testCharacterRepo{char: &domainchar.Character{ID: 1, Gold: 5}}
	features := &atomicTestFeatureRepo{testFeatureRepo: &testFeatureRepo{state: map[string]interface{}{}}}
	service := NewService(chars, features)
	result, err := service.Purchase(context.Background(), 1, 1, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if features.commits != 1 || features.gold != 4 || result.NewGold != 4 || chars.char.Gold != 4 {
		t.Fatalf("commits=%d dbGold=%d result=%+v charGold=%d", features.commits, features.gold, result, chars.char.Gold)
	}
}
