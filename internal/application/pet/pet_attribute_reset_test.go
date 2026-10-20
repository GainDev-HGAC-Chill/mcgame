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

func resetTestService(t *testing.T, petClass float64, itemTemplate int, spent, extra int) (*Service, *contractTestPetRepo, *bookSkillItemRepo) {
	t.Helper()
	petRepo := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{1: {
		ID: 1, CharacterID: 7, TemplateID: 301, Level: 50, GrowRate: 2.08,
		AttrPoints: 10, MaxAttrPoints: 10 + spent, DistributedAttrPoints: spent,
		AptStrengthEx: spent, AptEnergyEx: extra,
		Property: map[string]interface{}{"skill1": 789},
	}}}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		10: {ID: 10, CharacterID: 7, TemplateID: itemTemplate, SlotType: domainitem.SlotTypeBag, StackCount: 2},
	}}
	manager := gamedata.NewManager(nil, zap.NewNop())
	creatureData, err := json.Marshal(models.CreatureTemplate{ID: 301, ClassIds: petClass})
	if err != nil {
		t.Fatal(err)
	}
	itemData, err := json.Marshal(models.ItemTemplateTemplate{ID: int64(itemTemplate), Type: 507, UseType: 4, PropType: 1, ProplNum: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableCreature, []json.RawMessage{creatureData}); err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{itemData}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(petRepo, zap.NewNop())
	svc.SetGameDataManager(manager)
	svc.SetItemRepository(itemRepo)
	return svc, petRepo, itemRepo
}

func TestResetPetAttributesWithItemRefundsLegacyUnambiguousAllocation(t *testing.T) {
	svc, repo, items := resetTestService(t, 10, 1526, 30, 0)
	out, err := svc.ResetPetAttributesWithItem(context.Background(), 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	p, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.AttrPoints != 40 || p.DistributedAttrPoints != 0 || p.AptStrengthEx != 0 || p.AptEnergyEx != 0 || p.GrowRate != 2.08 || p.Property["skill1"] != 789 {
		t.Fatalf("incorrect reset: %+v", p)
	}
	if out.RemainingStack != 1 || items.items[10].StackCount != 1 {
		t.Fatalf("incorrect item consume: %+v", out)
	}
	if _, err := svc.ResetPetAttributesWithItem(context.Background(), 7, 1, 10); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("second reset: expected no point error, got %v", err)
	}
	if items.items[10].StackCount != 1 {
		t.Fatalf("replayed reset consumed pill")
	}
}

func TestResetPetAttributesWithItemRejectsMixedLegacyBonusesWithoutConsume(t *testing.T) {
	svc, repo, items := resetTestService(t, 10, 1526, 25, 5)
	if _, err := svc.ResetPetAttributesWithItem(context.Background(), 7, 1, 10); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("want invalid input, got %v", err)
	}
	p, _ := repo.FindByID(context.Background(), 1)
	if p.DistributedAttrPoints != 25 || p.AptStrengthEx != 25 || p.AptEnergyEx != 5 || items.items[10].StackCount != 2 {
		t.Fatalf("data changed on rejected reset: %+v %+v", p, items.items[10])
	}
}

func TestResetPetAttributesWithItemTracksAndKeepsOtherBonuses(t *testing.T) {
	svc, repo, _ := resetTestService(t, 10, 1526, 0, 5)
	p, _ := repo.FindByID(context.Background(), 1)
	if p.AllocateAttributePoints(4, 2, 0, 0, 0) != 6 {
		t.Fatal("allocation failed")
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	_, err := svc.ResetPetAttributesWithItem(context.Background(), 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := repo.FindByID(context.Background(), 1)
	if after.AttrPoints != 10 || after.AptEnergyEx != 5 || after.AptStrengthEx != 0 || after.AptAgilityEx != 0 || after.DistributedAttrPoints != 0 {
		t.Fatalf("tracked reset removed other bonuses: %+v", after)
	}
}

func TestResetPetAttributesWithItemRejectsWrongClassAndWrongItem(t *testing.T) {
	for _, tc := range []struct {
		class float64
		item  int
	}{{2, 1526}, {10, 275}} {
		svc, _, items := resetTestService(t, tc.class, tc.item, 5, 0)
		if _, err := svc.ResetPetAttributesWithItem(context.Background(), 7, 1, 10); !errors.Is(err, pkgerrors.ErrInvalidInput) {
			t.Fatalf("%+v: %v", tc, err)
		}
		if items.items[10].StackCount != 2 {
			t.Fatalf("%+v: item consumed", tc)
		}
	}
}

func TestResetPetAttributesWithItemRestoresPetWhenConsumptionFails(t *testing.T) {
	svc, repo, items := resetTestService(t, 10, 1526, 7, 0)
	items.updateStackErr = errors.New("db write failure")
	if _, err := svc.ResetPetAttributesWithItem(context.Background(), 7, 1, 10); err == nil {
		t.Fatal("wanted failure")
	}
	p, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.DistributedAttrPoints != 7 || p.AttrPoints != 10 || p.AptStrengthEx != 7 || items.items[10].StackCount != 2 {
		t.Fatalf("rollback failed: %+v", p)
	}
}
