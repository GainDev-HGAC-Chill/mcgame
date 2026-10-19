package pet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

func newPetWashTestService(t *testing.T, pillTemplateID int, proplNum float64) (*contractTestPetRepo, *bookSkillItemRepo, *Service) {
	t.Helper()
	repo, service := newFusionTestService(t)
	p, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	p.SetBinded(true)
	p.AptStrength = 5150
	p.AptAgility = 3350
	p.GrowRateAdd = 0.07
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		10: {ID: 10, CharacterID: 1, TemplateID: pillTemplateID,
			SlotType: domainitem.SlotTypeBag, StackCount: 2,
			Properties: map[string]interface{}{}},
	}}
	service.SetItemRepository(items)
	manager := newContractPetGameData(t, models.CreatureTemplate{
		ID: 100, AptStrength: 5150, AptAgility: 3350,
		AptStamina: 4000, AptIntelligence: 2000, AptEnergy: 2200,
	})
	tpl := models.ItemTemplateTemplate{ID: int64(pillTemplateID),
		Type: 507, UseType: 4, PropType: 2, ProplNum: proplNum}
	raw, err := json.Marshal(tpl)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	return repo, items, service
}

func TestPetXsdWashesOnlySelectedBaseAptitude(t *testing.T) {
	repo, items, service := newPetWashTestService(t, 275, 1)
	service.washRoll = func(min, max int) int {
		if min != 4120 || max != 6180 {
			t.Fatalf("roll bounds %d..%d", min, max)
		}
		return 5300
	}
	result, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Property != "aptStrength" || result.Before != 5150 || result.After != 5300 || result.AppliedValue != 5300 ||
		!result.Success || result.AptHigh || result.RemainingStack != 1 {
		t.Fatalf("result=%+v", result)
	}
	p, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.AptStrength != 5300 || p.AptAgility != 3350 || p.GrowRateAdd != 0.07 ||
		items.items[10].StackCount != 1 {
		t.Fatalf("pet=%+v item=%+v", p, items.items[10])
	}
}

func TestPetXsdOrdinaryLowerRollIsDiscarded(t *testing.T) {
	repo, items, service := newPetWashTestService(t, 275, 1)
	service.washRoll = func(min, max int) int { return 4800 }
	result, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Before != 5150 || result.After != 4800 || result.AppliedValue != 5150 ||
		!result.Success || items.items[10].StackCount != 1 {
		t.Fatalf("result=%+v", result)
	}
	p, err := repo.FindByID(context.Background(), 1)
	if err != nil || p.AptStrength != 5150 {
		t.Fatalf("pet=%+v err=%v", p, err)
	}
}

func TestPetXsdAdvancedCannotDecrease(t *testing.T) {
	repo, _, service := newPetWashTestService(t, 1652, 1)
	service.washRoll = func(min, max int) int {
		if min != 5151 || max != 6180 {
			t.Fatalf("advanced bounds %d..%d", min, max)
		}
		return max
	}
	result, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := repo.FindByID(context.Background(), 1)
	if p.AptStrength != 6180 || result.AptHigh || result.After != 6180 {
		t.Fatalf("pet=%+v result=%+v", p, result)
	}
}

func TestPetXsdAtMaxDoesNotConsumePill(t *testing.T) {
	repo, items, service := newPetWashTestService(t, 1652, 1)
	p, _ := repo.FindByID(context.Background(), 1)
	p.AptStrength = 6180
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	result, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AptHigh || result.RemainingStack != 2 || items.items[10].StackCount != 2 {
		t.Fatalf("result=%+v", result)
	}
}

func TestPetXsdRequiresBoundPet(t *testing.T) {
	repo, items, service := newPetWashTestService(t, 275, 1)
	p, _ := repo.FindByID(context.Background(), 1)
	p.SetBinded(false)
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	_, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err == nil || items.items[10].StackCount != 2 {
		t.Fatalf("unexpected err=%v", err)
	}
}

func TestPetXsdRejectsWrongItemCategory(t *testing.T) {
	_, items, service := newPetWashTestService(t, 275, 1)
	// A real user cannot substitute a different item template. The cache
	// lookup rejects absent/wrong types before any pet or inventory write.
	items.items[10].TemplateID = 9999
	_, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err == nil || items.items[10].StackCount != 2 {
		t.Fatalf("unexpected err=%v", err)
	}
}

func TestPetXsdItemConsumeFailureRestoresAptitude(t *testing.T) {
	repo, items, service := newPetWashTestService(t, 275, 1)
	items.updateStackErr = errors.New("db failure")
	service.washRoll = func(min, max int) int { return 5350 }
	if _, err := service.PetXsd(context.Background(), 1, 1, 10); err == nil {
		t.Fatal("expected error")
	}
	p, _ := repo.FindByID(context.Background(), 1)
	if p.AptStrength != 5150 || items.items[10].StackCount != 2 {
		t.Fatalf("pet=%+v item=%+v", p, items.items[10])
	}
}

func TestPetWashSlotMapsAllFiveAptitudes(t *testing.T) {
	pet := &domainpet.Pet{AptStrength: 100, AptAgility: 200, AptStamina: 300, AptIntelligence: 400, AptEnergy: 500}
	tpl := &models.CreatureTemplate{AptStrength: 1000, AptAgility: 2000, AptStamina: 3000, AptIntelligence: 4000, AptEnergy: 5000}
	cases := []struct {
		slot     int
		name     string
		current  int
		template float64
	}{
		{1, "aptStrength", 100, 1000},
		{2, "aptAgility", 200, 2000},
		{3, "aptStamina", 300, 3000},
		{4, "aptIntelligence", 400, 4000},
		{5, "aptEnergy", 500, 5000},
	}
	for _, tc := range cases {
		key, field, base := petWashSlot(pet, tpl, tc.slot)
		if key != tc.name || field == nil || *field != tc.current || base != tc.template {
			t.Fatalf("slot=%d got key=%s field=%v base=%v", tc.slot, key, field, base)
		}
	}
}

func TestPetXsdRejectsFakeItemOfSameCategory(t *testing.T) {
	_, items, service := newPetWashTestService(t, 5001, 1)
	_, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err == nil || items.items[10].StackCount != 2 {
		t.Fatalf("unexpected fake item result err=%v count=%d", err, items.items[10].StackCount)
	}
}

func TestPetXsdRejectsMismatchedSlot(t *testing.T) {
	_, items, service := newPetWashTestService(t, 275, 2)
	_, err := service.PetXsd(context.Background(), 1, 1, 10)
	if err == nil || items.items[10].StackCount != 2 {
		t.Fatalf("unexpected slot mismatch result err=%v count=%d", err, items.items[10].StackCount)
	}
}
