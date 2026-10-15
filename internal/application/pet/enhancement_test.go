package pet

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func TestUpgradeStarWithItemAddsConfiguredGrowthAndConsumesOne(t *testing.T) {
	for _, tc := range []struct {
		templateID int
		proplNum   float64
	}{
		{274, 5}, {280, 6}, {281, 7}, {282, 8}, {404, 9},
		{405, 10}, {406, 11}, {407, 12}, {408, 13}, {1651, 2},
	} {
		t.Run(strconv.Itoa(tc.templateID), func(t *testing.T) {
			petRepo := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{
				1: {ID: 1, CharacterID: 7, GrowRate: 1.5, GrowRateAdd: 0.05, Property: map[string]interface{}{}},
			}}
			itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
				10: {ID: 10, CharacterID: 7, TemplateID: tc.templateID, SlotType: domainitem.SlotTypeBag, StackCount: 2},
			}}
			service := NewService(petRepo, zap.NewNop())
			service.SetItemRepository(itemRepo)
			service.SetGameDataManager(petEnhancementGameData(t,
				models.ItemTemplateTemplate{ID: int64(tc.templateID), Type: 507, UseType: 4, PropType: 3, ProplNum: tc.proplNum},
			))

			result, err := service.UpgradeStarWithItem(context.Background(), 7, 1, 10)
			if err != nil {
				t.Fatal(err)
			}
			wantGrowth := 0.05 + tc.proplNum/100
			if result.Pet.UpgradeNum != 1 || math.Abs(result.Pet.GrowRateAdd-wantGrowth) > 1e-9 {
				t.Fatalf("upgraded pet = %+v, want growth %.2f", result.Pet, wantGrowth)
			}
			if result.RemainingStack != 1 || itemRepo.items[10].StackCount != 1 {
				t.Fatalf("item result = %+v, stored = %+v", result, itemRepo.items[10])
			}
			stored, err := petRepo.FindByID(context.Background(), 1)
			if err != nil || stored.UpgradeNum != 1 || math.Abs(stored.GrowRateAdd-wantGrowth) > 1e-9 {
				t.Fatalf("stored pet = %+v, err = %v", stored, err)
			}
		})
	}
}

func TestUpgradeStarWithItemRejectsNonEnhancementItemWithoutConsumption(t *testing.T) {
	petRepo := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{
		1: {ID: 1, CharacterID: 7, GrowRate: 1.5, Property: map[string]interface{}{}},
	}}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		10: {ID: 10, CharacterID: 7, TemplateID: 5351, SlotType: domainitem.SlotTypeBag, StackCount: 2},
	}}
	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(petEnhancementGameData(t,
		models.ItemTemplateTemplate{ID: 5351, Type: 550, UseType: 1},
	))

	_, err := service.UpgradeStarWithItem(context.Background(), 7, 1, 10)
	if !pkgerrors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
	if petRepo.pets[1].UpgradeNum != 0 || petRepo.pets[1].GrowRateAdd != 0 || itemRepo.items[10].StackCount != 2 {
		t.Fatalf("state changed: pet=%+v item=%+v", petRepo.pets[1], itemRepo.items[10])
	}
}

func TestPetStarSuccessRatesMatchClientDisplay(t *testing.T) {
	want := []int{100, 100, 90, 80, 70, 60, 50, 30, 30, 30, 30, 20}
	for targetStar, rate := range want {
		if got := petStarSuccessRates[targetStar+1]; got != rate {
			t.Fatalf("target star %d rate = %d, want %d", targetStar+1, got, rate)
		}
	}
}

func TestUpgradeStarWithItemFailureKeepsStarsBelowEight(t *testing.T) {
	service, petRepo, itemRepo := petEnhancementAttemptService(t, 7, 0, 2)
	service.starRoll = func(int) bool { return false }

	result, err := service.UpgradeStarWithItem(context.Background(), 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Pet.UpgradeNum != 7 || result.Pet.GrowRateAdd != 0.20 {
		t.Fatalf("result = %+v", result)
	}
	if petRepo.pets[1].UpgradeNum != 7 || itemRepo.items[10].StackCount != 1 {
		t.Fatalf("stored pet/item = %+v/%+v", petRepo.pets[1], itemRepo.items[10])
	}
}

func TestUpgradeStarWithItemFailureAtEightResetsStarsAndGrowthWithoutProtection(t *testing.T) {
	service, petRepo, _ := petEnhancementAttemptService(t, 8, 0, 2)
	service.starRoll = func(int) bool { return false }

	result, err := service.UpgradeStarWithItem(context.Background(), 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Pet.UpgradeNum != 0 || result.Pet.GrowRateAdd != 0 {
		t.Fatalf("result = %+v", result)
	}
	if petRepo.pets[1].UpgradeNum != 0 || petRepo.pets[1].GrowRateAdd != 0 {
		t.Fatalf("stored enhancement = stars %d/growth %.2f, want 0/0", petRepo.pets[1].UpgradeNum, petRepo.pets[1].GrowRateAdd)
	}
}

func TestUpgradeStarWithItemFailureAtEightConsumesProtectionAndKeepsStars(t *testing.T) {
	service, petRepo, itemRepo := petEnhancementAttemptService(t, 8, 2, 2)
	service.starRoll = func(int) bool { return false }

	result, err := service.UpgradeStarWithItem(context.Background(), 7, 1, 10, 11)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Pet.UpgradeNum != 8 || result.Pet.GrowRateAdd != 0.20 || result.SafeItemID != 11 || result.SafeRemaining != 1 {
		t.Fatalf("result = %+v", result)
	}
	if petRepo.pets[1].UpgradeNum != 8 || itemRepo.items[10].StackCount != 1 || itemRepo.items[11].StackCount != 1 {
		t.Fatalf("stored pet/items = %+v/%+v", petRepo.pets[1], itemRepo.items)
	}
}

func TestUpgradeStarWithItemUsesDisplayedTargetRate(t *testing.T) {
	service, _, _ := petEnhancementAttemptService(t, 11, 0, 2)
	gotRate := 0
	service.starRoll = func(rate int) bool {
		gotRate = rate
		return true
	}

	result, err := service.UpgradeStarWithItem(context.Background(), 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if gotRate != 20 || !result.Success || result.Pet.UpgradeNum != 12 {
		t.Fatalf("rate/result = %d/%+v", gotRate, result)
	}
}

func petEnhancementAttemptService(t *testing.T, stars int, safeStack, mainStack int) (*Service, *contractTestPetRepo, *bookSkillItemRepo) {
	t.Helper()
	petRepo := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{
		1: {ID: 1, CharacterID: 7, GrowRate: 1.56, GrowRateAdd: 0.20, UpgradeNum: stars, Property: map[string]interface{}{}},
	}}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		10: {ID: 10, CharacterID: 7, TemplateID: 274, SlotType: domainitem.SlotTypeBag, StackCount: mainStack},
	}}
	templates := []models.ItemTemplateTemplate{
		{ID: 274, Type: 507, UseType: 4, PropType: 3, ProplNum: 5},
	}
	if safeStack > 0 {
		itemRepo.items[11] = &domainitem.Item{ID: 11, CharacterID: 7, TemplateID: 2276, SlotType: domainitem.SlotTypeBag, StackCount: safeStack}
		templates = append(templates, models.ItemTemplateTemplate{ID: 2276, Type: 507, UseType: 4, PropType: 6, ProplNum: 1})
	}
	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(petEnhancementGameData(t, templates...))
	return service, petRepo, itemRepo
}

func petEnhancementGameData(t *testing.T, templates ...models.ItemTemplateTemplate) *gamedata.Manager {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	records := make([]json.RawMessage, 0, len(templates))
	for _, template := range templates {
		raw, err := json.Marshal(template)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, records); err != nil {
		t.Fatal(err)
	}
	return manager
}
