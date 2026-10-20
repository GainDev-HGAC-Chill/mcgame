package pet

import (
	"context"
	"errors"
	"math"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func resurrectionFixture(t *testing.T, quality float64, bound bool, pillTemplateID int64, stack int) (*Service, *contractTestPetRepo, *bookSkillItemRepo) {
	t.Helper()
	p := &domainpet.Pet{
		ID: 100, CharacterID: 3, TemplateID: 2290, Level: 92,
		AptStrength: 5000, AptAgility: 4800, AptStamina: 4500, AptIntelligence: 4000, AptEnergy: 3900,
		AptStrengthEx: 436, AptAgilityEx: 1, AptStaminaEx: 1, AptIntelligenceEx: 1, AptEnergyEx: 1,
		GrowRate: quality, GrowRateAdd: 0.03, UpgradeNum: 2,
		Property: map[string]interface{}{"skill1": 7026, "exampleFlag": "keep"},
	}
	p.SetBinded(bound)
	petRepo := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{}}
	if err := petRepo.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		55: {ID: 55, CharacterID: 3, TemplateID: int(pillTemplateID), SlotType: domainitem.SlotTypeBag, StackCount: stack, Properties: map[string]interface{}{}},
	}}
	s := NewService(petRepo, zap.NewNop())
	s.SetItemRepository(items)
	s.SetGameDataManager(petEnhancementGameData(t,
		models.ItemTemplateTemplate{ID: 1870, Type: 507, UseType: 4, PropType: 4, ProplNum: 1},
	))
	return s, petRepo, items
}

func TestPetResurrectionBounds(t *testing.T) {
	cases := []struct {
		rate     float64
		min, max int
		valid    bool
	}{
		{1.10, 0, 0, false},
		{1.11, 111, 130, true},
		{1.30, 111, 130, true},
		{1.31, 131, 160, true},
		{1.50, 131, 160, true},
		{1.60, 131, 160, true},
		{1.61, 161, 200, true},
		{2.00, 161, 200, true},
		{2.01, 201, 250, true},
		{2.08, 201, 250, true},
		{2.50, 201, 250, true},
		{2.51, 0, 0, false},
	}
	for _, tc := range cases {
		low, high, ok := petResurrectionBounds(tc.rate)
		if low != tc.min || high != tc.max || ok != tc.valid {
			t.Errorf("rate %v: got %d..%d ok=%v; want %d..%d ok=%v", tc.rate, low, high, ok, tc.min, tc.max, tc.valid)
		}
	}
}

func TestPetResurrectionIncreasesOnlyGrowRate(t *testing.T) {
	s, pets, items := resurrectionFixture(t, 2.08, true, 1870, 2)
	s.resurrectionRoll = func(min, max int) int {
		if min != 201 || max != 250 {
			t.Fatalf("unexpected range: %d..%d", min, max)
		}
		return 245
	}
	res, err := s.UseCultivationItem(context.Background(), 3, 100, 55)
	if err != nil {
		t.Fatal(err)
	}
	if res.Property != "growRate" || !res.Success || res.Before != 2.08 || res.After != 2.45 || res.RemainingStack != 1 {
		t.Fatalf("bad result: %#v", res)
	}
	if items.items[55].StackCount != 1 {
		t.Fatalf("pill was not consumed once: %d", items.items[55].StackCount)
	}
	stored, err := pets.FindByID(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GrowRate != 2.45 || stored.GrowRateAdd != 0.03 || stored.UpgradeNum != 2 ||
		stored.AptStrength != 5000 || stored.AptStrengthEx != 436 || stored.AptAgilityEx != 1 ||
		stored.AptStaminaEx != 1 || stored.AptIntelligenceEx != 1 || stored.AptEnergyEx != 1 ||
		stored.Property["skill1"] != 7026 || stored.Property["exampleFlag"] != "keep" {
		t.Fatalf("pet was modified incorrectly: %+v", stored)
	}
}

func TestPetResurrectionWorseRollKeepsOldQualityAndConsumesPill(t *testing.T) {
	s, pets, items := resurrectionFixture(t, 1.50, true, 1870, 2)
	s.resurrectionRoll = func(min, max int) int { return 136 }
	res, err := s.UseCultivationItem(context.Background(), 3, 100, 55)
	if err != nil {
		t.Fatal(err)
	}
	if res.Property != "growRate" || res.Success || res.Before != 1.5 || res.After != 1.36 || res.Pet.GrowRate != 1.5 {
		t.Fatalf("wrong worse-roll result: %+v", res)
	}
	stored, err := pets.FindByID(context.Background(), 100)
	if err != nil || stored.GrowRate != 1.5 || stored.GrowRateAdd != 0.03 || items.items[55].StackCount != 1 {
		t.Fatalf("worse roll modified pet or failed pill consume: %+v %+v %v", stored, items.items[55], err)
	}
}

func TestPetResurrectionRejectsInvalidInputWithoutConsumption(t *testing.T) {
	cases := []struct {
		name  string
		rate  float64
		bound bool
		tpl   int64
	}{
		{"unbound", 1.5, false, 1870},
		{"white", 1.1, true, 1870},
		{"above-max", 2.51, true, 1870},
		{"wrong-pill", 1.5, true, 275},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, pets, items := resurrectionFixture(t, tc.rate, tc.bound, tc.tpl, 2)
			_, err := s.UseCultivationItem(context.Background(), 3, 100, 55)
			if !pkgerrors.Is(err, pkgerrors.ErrInvalidInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
			stored, err := pets.FindByID(context.Background(), 100)
			if err != nil || stored.GrowRate != tc.rate || items.items[55].StackCount != 2 {
				t.Fatalf("invalid roll changed pet or item: %+v %+v %v", stored, items.items[55], err)
			}
		})
	}
}

func TestPetResurrectionRejectsBadTemplateMetadata(t *testing.T) {
	s, pets, items := resurrectionFixture(t, 1.5, true, 1870, 2)
	s.SetGameDataManager(petEnhancementGameData(t, models.ItemTemplateTemplate{ID: 1870, Type: 507, UseType: 4, PropType: 2, ProplNum: 1}))
	_, err := s.UseCultivationItem(context.Background(), 3, 100, 55)
	if !pkgerrors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("expected invalid template error, got %v", err)
	}
	stored, _ := pets.FindByID(context.Background(), 100)
	if stored.GrowRate != 1.5 || items.items[55].StackCount != 2 {
		t.Fatal("invalid template unexpectedly consumed pill")
	}
}

func TestPetResurrectionRollbackWhenInventoryFails(t *testing.T) {
	s, pets, items := resurrectionFixture(t, 1.5, true, 1870, 2)
	items.updateStackErr = errors.New("inventory unavailable")
	s.resurrectionRoll = func(min, max int) int { return 160 }
	_, err := s.UseCultivationItem(context.Background(), 3, 100, 55)
	if err == nil {
		t.Fatal("expected inventory error")
	}
	stored, err := pets.FindByID(context.Background(), 100)
	if err != nil || math.Abs(stored.GrowRate-1.5) > 1e-9 || items.items[55].StackCount != 2 {
		t.Fatalf("rollback failed: %+v %+v %v", stored, items.items[55], err)
	}
}
