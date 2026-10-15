package item

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type petGrowthTestService struct {
	pet    *domainpet.Pet
	amount int64
}

func (s *petGrowthTestService) ContractPets(context.Context, int64, []int) ([]*domainpet.Pet, error) {
	return nil, nil
}

func (s *petGrowthTestService) ContractPetsWithQuality(context.Context, int64, []int, int) ([]*domainpet.Pet, error) {
	return nil, nil
}

func (s *petGrowthTestService) GetPetDetail(_ context.Context, charID int64, petID int64) (*domainpet.Pet, error) {
	if s.pet == nil || s.pet.CharacterID != charID || s.pet.ID != petID {
		return nil, pkgerrors.ErrNotFound
	}
	copyPet := *s.pet
	return &copyPet, nil
}

func (s *petGrowthTestService) GainExperience(_ context.Context, charID int64, petID int64, amount int64) (*domainpet.Pet, bool, error) {
	if s.pet == nil || s.pet.CharacterID != charID || s.pet.ID != petID {
		return nil, false, pkgerrors.ErrNotFound
	}
	s.amount += amount
	copyPet := *s.pet
	copyPet.Experience += amount
	return &copyPet, true, nil
}

func TestPetGrowthMedicineExperienceMatchesFlashRegistry(t *testing.T) {
	want := map[int]int64{961: 30000, 962: 150000, 964: 300000, 1585: 300000, 1586: 2000000, 1587: 6400000, 1836: 12500000, 3339: 100000000}
	if len(petGrowthMedicineExperience) != len(want) {
		t.Fatalf("registry size = %d, want %d", len(petGrowthMedicineExperience), len(want))
	}
	for id, experience := range want {
		if petGrowthMedicineExperience[id] != experience {
			t.Fatalf("template %d experience = %d, want %d", id, petGrowthMedicineExperience[id], experience)
		}
	}
}

func TestPetGrowthMedicineAppliesQuantityAndUsesPetLevelGate(t *testing.T) {
	service := NewService(newItemServiceTestRepo(), zap.NewNop())
	petService := &petGrowthTestService{pet: &domainpet.Pet{ID: 9, CharacterID: 1, Level: 30, Property: map[string]interface{}{}}}
	service.SetPetService(petService)
	handler := &petGrowthOrPotionEffectHandler{itemService: service}
	tpl := &models.ItemTemplateTemplate{ID: 1586, Type: 501, UseType: 2, ReqLevel: 30}

	result, err := handler.Apply(context.Background(), &UseContext{TargetType: 2, PetID: 9, Quantity: 3}, &domainchar.Character{ID: 1, Level: 1}, &domainitem.Item{TemplateID: 1586}, tpl)
	if err != nil {
		t.Fatal(err)
	}
	if petService.amount != 6000000 {
		t.Fatalf("experience = %d, want 6000000", petService.amount)
	}
	if len(result.PetProgress) != 1 || !result.PetProgress[0].LeveledUp {
		t.Fatalf("progress = %#v", result.PetProgress)
	}
}

func TestPetGrowthMedicineRejectsWrongTargetAndLowLevelWithoutApplying(t *testing.T) {
	service := NewService(newItemServiceTestRepo(), zap.NewNop())
	petService := &petGrowthTestService{pet: &domainpet.Pet{ID: 9, CharacterID: 1, Level: 29}}
	service.SetPetService(petService)
	handler := &petGrowthOrPotionEffectHandler{itemService: service}
	tpl := &models.ItemTemplateTemplate{ID: 1586, Type: 501, UseType: 2, ReqLevel: 30}

	if _, err := handler.Apply(context.Background(), &UseContext{TargetType: 1, PetID: 9}, &domainchar.Character{ID: 1}, &domainitem.Item{}, tpl); !pkgerrors.Is(err, pkgerrors.ErrCannotUseItem) {
		t.Fatalf("wrong-target error = %v", err)
	}
	if _, err := handler.Apply(context.Background(), &UseContext{TargetType: 2, PetID: 9}, &domainchar.Character{ID: 1}, &domainitem.Item{}, tpl); !pkgerrors.Is(err, pkgerrors.ErrInsufficientLevel) {
		t.Fatalf("low-level error = %v", err)
	}
	if petService.amount != 0 {
		t.Fatalf("applied experience on rejected use: %d", petService.amount)
	}
}

func TestPetGrowthMedicineRejectsMaxLevelWithoutApplying(t *testing.T) {
	service := NewService(newItemServiceTestRepo(), zap.NewNop())
	petService := &petGrowthTestService{pet: &domainpet.Pet{ID: 9, CharacterID: 1, Level: domainpet.MaxPetLevel}}
	service.SetPetService(petService)
	handler := &petGrowthOrPotionEffectHandler{itemService: service}
	tpl := &models.ItemTemplateTemplate{ID: 3339, Type: 501, UseType: 2, ReqLevel: 80}

	if _, err := handler.Apply(context.Background(), &UseContext{TargetType: 2, PetID: 9}, &domainchar.Character{ID: 1}, &domainitem.Item{}, tpl); !pkgerrors.Is(err, pkgerrors.ErrCannotUseItem) {
		t.Fatalf("max-level error = %v", err)
	}
	if petService.amount != 0 {
		t.Fatalf("applied experience to max-level pet: %d", petService.amount)
	}
}
