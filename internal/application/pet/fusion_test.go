package pet

import (
	"context"
	"math"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"

	"go.uber.org/zap"
)

func TestFusePetsFailureConsumesFourMaterialsAndKeepsMain(t *testing.T) {
	repo, service := newFusionTestService(t)
	service.fusionRoll = func(int) bool { return false }
	result, err := service.FusePets(context.Background(), 1, 1, []int64{2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || len(result.ConsumedPetIDs) != 4 {
		t.Fatalf("unexpected result: %+v", result)
	}
	main, err := repo.FindByID(context.Background(), 1)
	if err != nil || main.GrowRateAdd != 0 {
		t.Fatalf("main pet changed on failure: pet=%+v err=%v", main, err)
	}
	if count, _ := repo.Count(context.Background(), 1); count != 1 {
		t.Fatalf("pet count = %d, want 1", count)
	}
	if _, err := service.FusePets(context.Background(), 1, 1, []int64{2, 3, 4, 5}); err == nil {
		t.Fatal("replayed fusion consumed missing pets")
	}
}

func TestFusePetsSuccessConsumesFourMaterialsAndUpdatesMainOnce(t *testing.T) {
	repo, service := newFusionTestService(t)
	service.fusionRoll = func(int) bool { return true }
	service.fusionGrowth = func(min, max int) int {
		if min != 111 || max != 130 {
			t.Fatalf("growth range=%d..%d", min, max)
		}
		return 124
	}
	mainPet, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	mainPet.GrowRateAdd = 0.11 // stale value written by the old fusion implementation
	if err := repo.Save(context.Background(), mainPet); err != nil {
		t.Fatal(err)
	}
	result, err := service.FusePets(context.Background(), 1, 1, []int64{2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || math.Abs(result.Pet.GrowRate-1.24) > 1e-9 || result.Pet.GrowRateAdd != 0 || domainpet.ColorByGrowRate(result.Pet.GrowRate) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if count, _ := repo.Count(context.Background(), 1); count != 1 {
		t.Fatalf("pet count = %d, want 1", count)
	}
}

func TestFusePetsSuccessPromotesBlueMainToPurple(t *testing.T) {
	repo, service := newFusionTestService(t)
	for id := int64(1); id <= 5; id++ {
		p, err := repo.FindByID(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		p.GrowRate = 1.5
		if err := repo.Save(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	service.fusionRoll = func(int) bool { return true }
	service.fusionGrowth = func(minimum, maximum int) int {
		if minimum != 161 || maximum != 200 {
			t.Fatalf("growth range=%d..%d, want 161..200", minimum, maximum)
		}
		return 195
	}

	result, err := service.FusePets(context.Background(), 1, 1, []int64{2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Pet.ID != 1 || result.Pet.GrowRate != 1.95 || domainpet.ColorByGrowRate(result.Pet.GrowRate) != 3 {
		t.Fatalf("unexpected blue-to-purple result: %+v", result)
	}
	if count, _ := repo.Count(context.Background(), 1); count != 1 {
		t.Fatalf("pet count = %d, want only main pet", count)
	}
}

func TestFusePetsRejectsDifferentTemplateOrTierWithoutConsumption(t *testing.T) {
	for _, mutate := range []func(*domainpet.Pet){
		func(p *domainpet.Pet) { p.TemplateID = 101 },
		func(p *domainpet.Pet) { p.GrowRate = 1.2 },
	} {
		repo, service := newFusionTestService(t)
		material, err := repo.FindByID(context.Background(), 2)
		if err != nil {
			t.Fatal(err)
		}
		mutate(material)
		if err := repo.Save(context.Background(), material); err != nil {
			t.Fatal(err)
		}
		if _, err := service.FusePets(context.Background(), 1, 1, []int64{2, 3, 4, 5}); err == nil {
			t.Fatal("expected incompatible material error")
		}
		if count, _ := repo.Count(context.Background(), 1); count != 5 {
			t.Fatalf("pet count = %d, want 5", count)
		}
	}
}

func TestFusePetsRejectsMaximumTier(t *testing.T) {
	repo, service := newFusionTestService(t)
	for id := int64(1); id <= 5; id++ {
		pet, err := repo.FindByID(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		pet.GrowRate = 2.1
		if err := repo.Save(context.Background(), pet); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.FusePets(context.Background(), 1, 1, []int64{2, 3, 4, 5}); err == nil {
		t.Fatal("expected maximum-tier rejection")
	}
	if count, _ := repo.Count(context.Background(), 1); count != 5 {
		t.Fatalf("pet count = %d, want 5", count)
	}
}

func TestPetFusionSuccessRatesMatchClientTiers(t *testing.T) {
	for _, spec := range []struct {
		grow float64
		rate int
	}{
		{1.0, 100},
		{1.2, 80},
		{1.5, 50},
		{1.9, 30},
		{2.1, 0},
	} {
		if got := petFusionSuccessRate(&domainpet.Pet{GrowRate: spec.grow}); got != spec.rate {
			t.Fatalf("grow %.2f rate=%d want=%d", spec.grow, got, spec.rate)
		}
	}
}

func newFusionTestService(t *testing.T) (*contractTestPetRepo, *Service) {
	t.Helper()
	repo := &contractTestPetRepo{pets: make(map[int64]*domainpet.Pet)}
	for id := int64(1); id <= 5; id++ {
		pet := &domainpet.Pet{ID: id, CharacterID: 1, TemplateID: 100, GrowRate: 0.8, Property: map[string]interface{}{}}
		if err := repo.Save(context.Background(), pet); err != nil {
			t.Fatal(err)
		}
	}
	return repo, NewService(repo, zap.NewNop())
}
