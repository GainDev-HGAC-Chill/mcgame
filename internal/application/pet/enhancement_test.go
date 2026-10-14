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
