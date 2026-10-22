package specialtraining

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
)

type trainingStateRepo struct {
	mu      sync.Mutex
	state   *domainfeature.CharacterFeatureState
	saveErr error
}

func (r *trainingStateRepo) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == nil {
		return nil, nil
	}
	copy := *r.state
	return []*domainfeature.CharacterFeatureState{&copy}, nil
}

func (r *trainingStateRepo) UpsertCharacterFeatureState(_ context.Context, state *domainfeature.CharacterFeatureState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	copy := *state
	r.state = &copy
	return nil
}

type trainingCharacterRepo struct {
	mu   sync.Mutex
	char *domainchar.Character
}

func (r *trainingCharacterRepo) FindByID(context.Context, int64) (*domainchar.Character, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *r.char
	return &copy, nil
}

func (r *trainingCharacterRepo) Update(_ context.Context, char *domainchar.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *char
	r.char = &copy
	return nil
}

func TestSeasonPayloadUsesApprovedFixedWindow(t *testing.T) {
	service, _, _ := newTrainingService(1000)
	payload, err := service.GetData(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if payload["start"] != "2026-10-22" || payload["end"] != "2026-12-31" || payload["days"] != 70 {
		t.Fatalf("payload=%v", payload)
	}
	flag := payload["flag"].(map[string]interface{})
	if flag["lev"] != 1 || flag["exp"] != 0 || flag["p"] != 0 || flag["awd1"] != 0 || flag["awd2"] != 0 {
		t.Fatalf("flag=%v", flag)
	}
}

func TestPremiumAndEXPPurchasesUseAuthoritativeGoldPrices(t *testing.T) {
	service, _, chars := newTrainingService(2000)
	premium, err := service.OpenPremium(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if premium.GoldSpent != 680 || premium.Gold != 1320 || premium.Payload["flag"].(map[string]interface{})["p"] != 1 {
		t.Fatalf("premium=%+v", premium)
	}
	if _, err := service.OpenPremium(context.Background(), 1); !errors.Is(err, ErrAlreadyPremium) {
		t.Fatalf("duplicate premium error=%v", err)
	}
	bought, err := service.BuyEXP(context.Background(), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	flag := bought.Payload["flag"].(map[string]interface{})
	if bought.GoldSpent != 600 || bought.Gold != 720 || flag["lev"] != 3 || flag["exp"] != 0 {
		t.Fatalf("bought=%+v flag=%v", bought, flag)
	}
	if chars.char.Gold != 720 {
		t.Fatalf("gold=%d", chars.char.Gold)
	}
}

func TestDailyAndWeeklyTaskClaimAndVietnamResets(t *testing.T) {
	service, _, _ := newTrainingService(0)
	now := approvedTrainingConfig().Start
	service.SetClock(func() time.Time { return now })
	if err := service.RecordTask(context.Background(), 1, 1, 37, 3); !errors.Is(err, ErrTaskUnavailable) {
		t.Fatalf("closed daily error=%v", err)
	}
	if _, err := service.OpenDailyTasks(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if err := service.RecordTask(context.Background(), 1, 1, 37, 3); err != nil {
		t.Fatal(err)
	}
	payload, err := service.ClaimTaskEXP(context.Background(), 1, 1, 37)
	if err != nil {
		t.Fatal(err)
	}
	flag := payload["flag"].(map[string]interface{})
	if flag["exp"] != 120 || flag["dd"].(map[string]interface{})["37"] != "exped" {
		t.Fatalf("daily flag=%v", flag)
	}
	if err := service.RecordTask(context.Background(), 1, 2, 1, 5); err != nil {
		t.Fatal(err)
	}
	payload, err = service.ClaimTaskEXP(context.Background(), 1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if payload["flag"].(map[string]interface{})["exp"] != 400 {
		t.Fatalf("weekly payload=%v", payload)
	}
	now = now.Add(24 * time.Hour)
	payload, err = service.GetData(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := payload["flag"].(map[string]interface{})["dd"]; exists {
		t.Fatalf("daily tasks did not close at reset: %v", payload)
	}
}

func TestRewardReservationIsSequentialAndPremiumGated(t *testing.T) {
	service, _, _ := newTrainingService(1000)
	if _, _, _, err := service.ReserveNextReward(context.Background(), 1, true); !errors.Is(err, ErrRewardUnavailable) {
		t.Fatalf("premium gate error=%v", err)
	}
	reward, level, payload, err := service.ReserveNextReward(context.Background(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if reward != (Reward{ItemID: 2333, Count: 2}) || level != 1 || payload["flag"].(map[string]interface{})["awd1"] != 1 {
		t.Fatalf("reward=%+v level=%d payload=%v", reward, level, payload)
	}
	if _, _, _, err := service.ReserveNextReward(context.Background(), 1, false); !errors.Is(err, ErrRewardUnavailable) {
		t.Fatalf("duplicate reward error=%v", err)
	}
}

func TestGoldCompensationAndConcurrentPurchases(t *testing.T) {
	service, states, chars := newTrainingService(600)
	states.saveErr = errors.New("save failed")
	if _, err := service.BuyEXP(context.Background(), 1, 1); err == nil {
		t.Fatal("purchase unexpectedly succeeded")
	}
	if chars.char.Gold != 600 {
		t.Fatalf("gold after compensation=%d", chars.char.Gold)
	}
	states.saveErr = nil
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.BuyEXP(context.Background(), 1, 1)
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
	payload, err := service.GetData(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if chars.char.Gold != 0 || payload["flag"].(map[string]interface{})["lev"] != 3 {
		t.Fatalf("gold=%d payload=%v", chars.char.Gold, payload)
	}
}

func TestRewardMapsMatchSeventyCatalogItems(t *testing.T) {
	if len(FreeRewards) != 71 || len(PremiumRewards) != 71 {
		t.Fatalf("reward lengths=%d/%d", len(FreeRewards), len(PremiumRewards))
	}
	content, err := os.ReadFile("../../../docs/database/game_data/TBL_ITEM_TEMPLATE.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	items := make(map[int]bool, len(catalog.Records))
	for _, row := range catalog.Records {
		id, err := strconv.Atoi(row.ID)
		if err == nil {
			items[id] = true
		}
	}
	for level := 1; level <= MaxLevel; level++ {
		for _, reward := range []Reward{FreeRewards[level], PremiumRewards[level]} {
			if !items[reward.ItemID] || reward.Count <= 0 {
				t.Fatalf("invalid level %d reward %+v", level, reward)
			}
		}
	}
}

func newTrainingService(gold int64) (*Service, *trainingStateRepo, *trainingCharacterRepo) {
	states := &trainingStateRepo{}
	chars := &trainingCharacterRepo{char: &domainchar.Character{ID: 1, Gold: gold}}
	service := NewService(states, chars, approvedTrainingConfig())
	service.SetClock(func() time.Time { return approvedTrainingConfig().Start })
	return service, states, chars
}

func approvedTrainingConfig() Config {
	location := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	return Config{
		Key: "2026-10-22_2026-12-31", Start: time.Date(2026, 10, 22, 0, 0, 0, 0, location),
		End: time.Date(2026, 12, 31, 0, 0, 0, 0, location), Location: location,
	}
}
