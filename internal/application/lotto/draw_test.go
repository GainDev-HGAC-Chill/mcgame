// Open-sourced by BaoLT

package lotto

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type fakeGameData struct {
	byType map[int][]*models.PlanTemplate
}

func (g *fakeGameData) PlansByType(t int) []*models.PlanTemplate { return g.byType[t] }

type fakeChars struct {
	char *domainchar.Character
}

func (c *fakeChars) FindByID(ctx context.Context, id int64) (*domainchar.Character, error) {
	c.char.ID = id
	return c.char, nil
}

func (c *fakeChars) SyncPersistedGold(charID, gold int64) {
	c.char.Gold = gold
}

func newDrawService(gold int64) (*Service, *fakeChars) {
	svc := NewService(newFakeRepo(), zap.NewNop())
	svc.SetGameData(&fakeGameData{byType: map[int][]*models.PlanTemplate{
		1: {{ID: 101, Ti: 29, Ii: 5, N: 1, Q: 0, B: 1, R: 1.0}},
		2: {{ID: 201, Ti: 19, Ii: 900, N: 1, Q: 2, B: 0, R: 0.5}},
		3: {{ID: 301, Ti: 29, Ii: 7, N: 3, Q: 0, B: 1, R: 0.2}},
	}})
	chars := &fakeChars{char: &domainchar.Character{Gold: gold}}
	svc.SetCharRepository(chars)
	return svc, chars
}

const drawChar = int64(1)

func TestDraw_FreeType1SingleUntilQuotaThenCharges(t *testing.T) {
	svc, chars := newDrawService(1000)
	ctx := context.Background()

	for i := 0; i < freeWishDailyMax; i++ {
		if _, err := svc.Draw(ctx, drawChar, 1, 1); err != nil {
			t.Fatalf("free draw %d: %v", i, err)
		}
	}
	if chars.char.Gold != 1000 {
		t.Fatalf("gold spent during free quota: %d, want 1000", chars.char.Gold)
	}

	if _, err := svc.Draw(ctx, drawChar, 1, 1); err != nil {
		t.Fatalf("paid draw after quota: %v", err)
	}
	if chars.char.Gold != 1000-lottoGoldCost[1][1] {
		t.Fatalf("gold after first paid single = %d, want %d", chars.char.Gold, 1000-lottoGoldCost[1][1])
	}

	lottoLen, _ := svc.BagLengths(ctx, drawChar)
	if lottoLen != freeWishDailyMax+1 {
		t.Fatalf("bag len = %d, want %d", lottoLen, freeWishDailyMax+1)
	}
}

func TestDraw_InvalidNumRejected(t *testing.T) {
	svc, chars := newDrawService(2000)
	if _, err := svc.Draw(context.Background(), drawChar, 2, 0); err != ErrLottoInvalidNum {
		t.Fatalf("num 0 should be ErrLottoInvalidNum, got %v", err)
	}
	if _, err := svc.Draw(context.Background(), drawChar, 2, 3); err != ErrLottoInvalidNum {
		t.Fatalf("num 3 should be ErrLottoInvalidNum, got %v", err)
	}
	if chars.char.Gold != 2000 {
		t.Fatalf("invalid-num draw mutated gold: %d", chars.char.Gold)
	}
}

func TestDraw_ChargesBundleAndDepositsNum(t *testing.T) {
	svc, chars := newDrawService(2000)
	ctx := context.Background()

	bag, err := svc.Draw(ctx, drawChar, 2, 10)
	if err != nil {
		t.Fatalf("Draw type2 x10: %v", err)
	}
	if len(bag) != 10 {
		t.Fatalf("x10 deposited %d entries, want 10", len(bag))
	}
	if chars.char.Gold != 2000-lottoGoldCost[2][10] {
		t.Fatalf("gold after x10 = %d, want %d", chars.char.Gold, 2000-lottoGoldCost[2][10])
	}
	if bag[0].(map[string]any)["ti"] != "19" {
		t.Fatalf("type2 entry ti = %v, want 19", bag[0].(map[string]any)["ti"])
	}
}

func TestDraw_InsufficientGoldDoesNotDeposit(t *testing.T) {
	svc, chars := newDrawService(10)
	ctx := context.Background()

	if _, err := svc.Draw(ctx, drawChar, 3, 1); err != ErrLottoInsufficient {
		t.Fatalf("expected ErrLottoInsufficient, got %v", err)
	}
	if chars.char.Gold != 10 {
		t.Fatalf("gold mutated on failed draw: %d", chars.char.Gold)
	}
	lottoLen, _ := svc.BagLengths(ctx, drawChar)
	if lottoLen != 0 {
		t.Fatalf("failed draw deposited %d entries, want 0", lottoLen)
	}
}

func TestDraw_InvalidTypeRejected(t *testing.T) {
	svc, _ := newDrawService(1000)
	ctx := context.Background()
	if _, err := svc.Draw(ctx, drawChar, 4, 1); err != ErrLottoInvalidType {
		t.Fatalf("type 4 should be ErrLottoInvalidType, got %v", err)
	}
	if _, err := svc.Draw(ctx, drawChar, 0, 1); err != ErrLottoInvalidType {
		t.Fatalf("type 0 should be ErrLottoInvalidType, got %v", err)
	}
}

func TestGetLottoData_GridByType(t *testing.T) {
	svc, _ := newDrawService(0)
	data, err := svc.GetLottoData(context.Background(), drawChar)
	if err != nil {
		t.Fatalf("GetLottoData: %v", err)
	}
	award, ok := data["lottoAward"].([]any)
	if !ok || len(award) != lottoDisplayTypes {
		t.Fatalf("lottoAward shape wrong: %#v", data["lottoAward"])
	}
	type1 := award[0].([]any)
	if len(type1) != 1 || type1[0] != int64(101) {
		t.Fatalf("type1 grid = %#v, want [101]", type1)
	}
	if _, ok := data["highestAwardArr"].([]any); !ok {
		t.Fatalf("highestAwardArr missing/wrong: %#v", data["highestAwardArr"])
	}
}

func TestAddToBag_RejectsPastCap(t *testing.T) {
	svc := NewService(newFakeRepo(), zap.NewNop())
	ctx := context.Background()
	state, _ := svc.loadState(ctx, drawChar)
	for i := 0; i < lottoBagCap; i++ {
		state.Lotto = append(state.Lotto, LottoEntry{Ti: "29", Ii: "5", N: 1, B: 1, Q: "0"})
	}
	if err := svc.saveState(ctx, drawChar, state); err != nil {
		t.Fatalf("seed full bag: %v", err)
	}
	if err := svc.AddToBag(ctx, drawChar, BagLotto, LottoEntry{Ti: "29", Ii: "5", N: 1, B: 1, Q: "0"}); err != ErrLottoBagFull {
		t.Fatalf("AddToBag past cap = %v, want ErrLottoBagFull", err)
	}
}

func TestDraw_FreeQuotaResetsOnNewDay(t *testing.T) {
	svc, chars := newDrawService(1000)
	ctx := context.Background()
	for i := 0; i < freeWishDailyMax; i++ {
		if _, err := svc.Draw(ctx, drawChar, 1, 1); err != nil {
			t.Fatalf("free draw: %v", err)
		}
	}
	state, _ := svc.loadState(ctx, drawChar)
	state.FreeWishDay = "2000-01-01"
	_ = svc.saveState(ctx, drawChar, state)

	if _, err := svc.Draw(ctx, drawChar, 1, 1); err != nil {
		t.Fatalf("draw after day rollover: %v", err)
	}
	if chars.char.Gold != 1000 {
		t.Fatalf("quota did not reset; gold spent: %d", chars.char.Gold)
	}
}

func TestDraw_WishPetCreatesCanonicalFacts(t *testing.T) {
	svc, _ := newDrawService(2000)
	svc.SetGameData(&fakeGameData{byType: map[int][]*models.PlanTemplate{
		2: {
			{ID: 201, Ti: 12, Ii: 501, N: 1, Q: 12, R: 1},
			{ID: 202, Ti: 12, Ii: 502, N: 1, Q: 15, R: 1},
			{ID: 203, Ti: 12, Ii: 503, N: 1, Q: 19, R: 1},
			{ID: 204, Ti: 12, Ii: 504, N: 1, Q: 23, R: 1},
		},
	}})
	plans := svc.gameData.PlansByType(2)
	facts, err := wishPetFacts(drawChar, "8a89bb8f-9323-4f58-96dc-035fcdadf479", plans, time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("wishPetFacts() error = %v", err)
	}
	if len(facts) != 4 {
		t.Fatalf("facts = %d, want 4", len(facts))
	}
	wantRates := []float64{1.2, 1.5, 1.9, 2.3}
	wantColors := []int{1, 2, 3, 4}
	for i, fact := range facts {
		var payload domainachievement.WishPetDrawnFactPayload
		if err := json.Unmarshal(fact.Payload, &payload); err != nil {
			t.Fatalf("decode fact %d: %v", i, err)
		}
		if payload.GrowRate != wantRates[i] || payload.ColorTier != wantColors[i] || payload.Ordinal != i {
			t.Fatalf("fact %d payload = %#v", i, payload)
		}
	}
}

func TestDraw_CommitFailureDoesNotMutateGoldOrState(t *testing.T) {
	svc, chars := newDrawService(2000)
	repo := svc.featureRepo.(*fakeFeatureRepo)
	repo.commitErr = errors.New("injected commit failure")
	if _, err := svc.Draw(context.Background(), drawChar, 2, 1); err == nil {
		t.Fatal("Draw() error = nil, want commit failure")
	}
	if chars.char.Gold != 2000 {
		t.Fatalf("gold = %d, want 2000", chars.char.Gold)
	}
	lottoLen, _ := svc.BagLengths(context.Background(), drawChar)
	if lottoLen != 0 || len(repo.facts) != 0 {
		t.Fatalf("failed draw persisted bag=%d facts=%d", lottoLen, len(repo.facts))
	}
}
