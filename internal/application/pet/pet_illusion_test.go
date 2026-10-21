package pet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type atomicIllusionPetRepo struct {
	*contractTestPetRepo
	err        error
	called     bool
	updateMain bool
}

func (r *atomicIllusionPetRepo) ApplyIllusion(ctx context.Context, mainPet *domainpet.Pet, consumedPetID, luckItemID int64, luckItemCount int, updateMain bool) (int, error) {
	r.called = true
	r.updateMain = updateMain
	if r.err != nil {
		return 0, r.err
	}
	if updateMain {
		if err := r.Save(ctx, mainPet); err != nil {
			return 0, err
		}
	}
	if err := r.Delete(ctx, consumedPetID); err != nil {
		return 0, err
	}
	return 0, nil
}

func newPetIllusionTestService(t *testing.T) (*Service, *contractTestPetRepo, *bookSkillItemRepo) {
	t.Helper()
	pets := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{
		1: {
			ID: 1, CharacterID: 7, TemplateID: 301, GrowRate: 1.8,
			AptStrength: 1000, AptAgility: 1200, AptStamina: 1300, AptIntelligence: 1400, AptEnergy: 1500,
			AptAgilityEx: 7, Property: map[string]interface{}{"binded": 1},
		},
		2: {
			ID: 2, CharacterID: 7, TemplateID: 302, GrowRate: 1.9,
			AptStrength: 3400, AptAgility: 4800, AptStamina: 5300, AptIntelligence: 2150, AptEnergy: 2260,
			Property: map[string]interface{}{},
		},
	}}
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		10: {ID: 10, CharacterID: 7, TemplateID: 2326, SlotType: domainitem.SlotTypeBag, StackCount: 5},
	}}
	manager := gamedata.NewManager(nil, zap.NewNop())
	cache := manager.GetCache()
	if err := cache.LoadTable(models.TableCreature, []json.RawMessage{
		json.RawMessage(`{"id":301,"name":"Main","class_id":2,"class_ids":10,"q_level":3,"apt_strength":1000,"apt_agility":1200,"apt_stamina":1300,"apt_intelligence":1400,"apt_energy":1500}`),
		json.RawMessage(`{"id":302,"name":"Cost","class_id":2,"class_ids":10,"q_level":3,"apt_strength":3400,"apt_agility":4800,"apt_stamina":5300,"apt_intelligence":2150,"apt_energy":2260}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.LoadTable(models.TableItemTemplate, []json.RawMessage{
		json.RawMessage(`{"id":2326,"name":"Đá May Mắn Cam","type":507,"use_type":4,"prop_type":7}`),
	}); err != nil {
		t.Fatal(err)
	}
	service := NewService(pets, zap.NewNop())
	service.SetItemRepository(items)
	service.SetGameDataManager(manager)
	return service, pets, items
}

func TestPetEatSuccessUsesFlashRateAndResponseValue(t *testing.T) {
	service, pets, items := newPetIllusionTestService(t)
	service.illusionRoll = func(rate int) bool {
		if rate != 90 {
			t.Fatalf("success rate = %d, want 90", rate)
		}
		return true
	}
	service.illusionPercent = func(minimum, maximum int) int {
		if minimum != 10 || maximum != 15 {
			t.Fatalf("percent range = %d..%d", minimum, maximum)
		}
		return 12
	}

	result, err := service.PetEat(context.Background(), 7, 1, 2, 10, 2, domainpet.AptTypeAgility)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.PropertyName != "aptAgilityEx" || result.Value != 583 || result.ConsumedLuckCount != 2 || result.RemainingLuck != 3 {
		t.Fatalf("result = %+v", result)
	}
	stored, err := pets.FindByID(context.Background(), 1)
	if err != nil || stored.AptAgilityEx != 583 {
		t.Fatalf("stored main = %+v, err=%v", stored, err)
	}
	if _, err := pets.FindByID(context.Background(), 2); !errors.Is(err, pkgerrors.ErrNotFound) {
		t.Fatalf("cost pet still exists: %v", err)
	}
	if items.items[10].StackCount != 3 {
		t.Fatalf("luck stack = %d, want 3", items.items[10].StackCount)
	}
}

func TestPetEatFailureConsumesCostAndLuckButKeepsMain(t *testing.T) {
	service, pets, items := newPetIllusionTestService(t)
	service.illusionRoll = func(rate int) bool {
		if rate != 70 {
			t.Fatalf("success rate = %d, want 70", rate)
		}
		return false
	}

	result, err := service.PetEat(context.Background(), 7, 1, 2, 10, 1, domainpet.AptTypeAgility)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Value != 7 || result.ConsumedPetID != 2 || result.ConsumedLuckCount != 1 {
		t.Fatalf("result = %+v", result)
	}
	stored, err := pets.FindByID(context.Background(), 1)
	if err != nil || stored.AptAgilityEx != 7 {
		t.Fatalf("main changed: %+v, err=%v", stored, err)
	}
	if _, err := pets.FindByID(context.Background(), 2); !errors.Is(err, pkgerrors.ErrNotFound) {
		t.Fatalf("cost pet still exists: %v", err)
	}
	if items.items[10].StackCount != 4 {
		t.Fatalf("luck stack = %d, want 4", items.items[10].StackCount)
	}
}

func TestSetPetIllusionAptitudeReplacesOnlyTrackedBonus(t *testing.T) {
	p := &domainpet.Pet{AptAgilityEx: 7, Property: map[string]interface{}{}}
	if value, ok := setPetIllusionAptitude(p, domainpet.AptTypeAgility, 100); !ok || value != 107 {
		t.Fatalf("first value=%d ok=%v", value, ok)
	}
	if value, ok := setPetIllusionAptitude(p, domainpet.AptTypeAgility, 50); !ok || value != 57 {
		t.Fatalf("replacement value=%d ok=%v", value, ok)
	}
}

func TestPetEatRejectsBelowPurpleWithoutConsumption(t *testing.T) {
	service, pets, items := newPetIllusionTestService(t)
	main, _ := pets.FindByID(context.Background(), 1)
	main.GrowRate = 1.5
	_ = pets.Save(context.Background(), main)

	if _, err := service.PetEat(context.Background(), 7, 1, 2, 10, 1, domainpet.AptTypeAgility); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
	if count, _ := pets.Count(context.Background(), 7); count != 2 || items.items[10].StackCount != 5 {
		t.Fatalf("invalid attempt consumed data: pets=%d stack=%d", count, items.items[10].StackCount)
	}
}

func TestPetEatAtomicMutationErrorLeavesAllStateUnchanged(t *testing.T) {
	service, pets, items := newPetIllusionTestService(t)
	mutationErr := errors.New("atomic mutation failed")
	atomicRepo := &atomicIllusionPetRepo{contractTestPetRepo: pets, err: mutationErr}
	service.petRepo = atomicRepo
	service.illusionRoll = func(int) bool { return true }
	service.illusionPercent = func(int, int) int { return 12 }

	if _, err := service.PetEat(context.Background(), 7, 1, 2, 10, 5, domainpet.AptTypeAgility); !errors.Is(err, mutationErr) {
		t.Fatalf("error = %v, want atomic mutation error", err)
	}
	if !atomicRepo.called || !atomicRepo.updateMain {
		t.Fatalf("atomic mutation called=%v updateMain=%v", atomicRepo.called, atomicRepo.updateMain)
	}
	mainPet, err := pets.FindByID(context.Background(), 1)
	if err != nil || mainPet.AptAgilityEx != 7 {
		t.Fatalf("main pet changed: pet=%+v err=%v", mainPet, err)
	}
	if _, err := pets.FindByID(context.Background(), 2); err != nil {
		t.Fatalf("cost pet changed: %v", err)
	}
	if items.items[10].StackCount != 5 {
		t.Fatalf("luck stack = %d, want 5", items.items[10].StackCount)
	}
}
