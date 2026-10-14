package item

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func capacityTestService(t *testing.T, templateID, stack int) (*Service, *itemServiceTestRepo) {
	t.Helper()
	repo := newItemServiceTestRepo()
	repo.items[88] = &domainitem.Item{ID: 88, CharacterID: 99, TemplateID: templateID, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: stack}
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, err := json.Marshal(models.ItemTemplateTemplate{ID: int64(templateID), Kind: 5, Type: 500, UseType: 1, ReqLevel: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	svc.SetGameDataManager(manager)
	return svc, repo
}

func TestUseCapacityBookExpandsExactlyOneAndConsumesOne(t *testing.T) {
	for _, spec := range []struct {
		id    int
		field string
	}{
		{324, "bankSlotNum"}, {325, "bagSlotNum"}, {326, "petMaxNum"},
	} {
		t.Run(spec.field, func(t *testing.T) {
			svc, repo := capacityTestService(t, spec.id, 2)
			char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BagSlots: 30, BankSlotNum: 1, BankSlots: 30, PetMaxNum: 6, PetSlots: 6, Gold: 50, GoldBind: 60, Money: 70}
			want := *char
			next := 2
			switch spec.id {
			case 324:
				want.SetBankSlotNum(2)
			case 325:
				want.SetBagSlotNum(2)
			case 326:
				want.PetMaxNum, want.PetSlots, next = 7, 7, 7
			}
			it, result, err := svc.UseItem(context.Background(), char, 88, &UseContext{TargetType: 1, Quantity: 100})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*char, want) {
				t.Fatalf("unexpected character change: got %+v want %+v", *char, want)
			}
			if it.StackCount != 1 || repo.items[88].StackCount != 1 {
				t.Fatal("single use did not consume exactly one book")
			}
			if !reflect.DeepEqual(result["capacity_updates"], map[string]interface{}{spec.field: next}) {
				t.Fatalf("capacity update = %v", result)
			}
		})
	}
}

func TestUseCapacityBookLimitsAndLegacyFields(t *testing.T) {
	for _, spec := range []struct {
		name string
		id   int
		char domainchar.Character
		want int
	}{
		{"bank last tab", 324, domainchar.Character{BankSlotNum: 4, BankSlots: 120}, 5},
		{"bag last tab", 325, domainchar.Character{BagSlotNum: 6, BagSlots: 180}, 7},
		{"pet past obsolete description cap", 326, domainchar.Character{PetMaxNum: 84, PetSlots: 84}, 85},
		{"pet last slot", 326, domainchar.Character{PetMaxNum: 179, PetSlots: 179}, 180},
		{"legacy bank", 324, domainchar.Character{BankSlots: 60}, 3},
		{"legacy bag", 325, domainchar.Character{BagSlots: 60}, 3},
		{"legacy pet", 326, domainchar.Character{PetSlots: 10}, 11},
	} {
		t.Run(spec.name, func(t *testing.T) {
			svc, repo := capacityTestService(t, spec.id, 1)
			char := spec.char
			char.ID, char.Level = 99, 1
			_, result, err := svc.UseItem(context.Background(), &char, 88, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(repo.items) != 0 || result["remaining"] != 0 {
				t.Fatal("last book not deleted")
			}
			got := char.BankSlotNum
			switch spec.id {
			case 324:
				if char.BankSlots != spec.want*30 {
					t.Fatal("bank slot aliases diverged")
				}
			case 325:
				got = char.BagSlotNum
				if char.BagSlots != spec.want*30 {
					t.Fatal("bag slot aliases diverged")
				}
			case 326:
				got = char.PetMaxNum
				if char.PetSlots != spec.want {
					t.Fatal("pet slot aliases diverged")
				}
			}
			if got != spec.want {
				t.Fatalf("capacity = %d want %d", got, spec.want)
			}
		})
	}
}

func TestUseCapacityBookRejectsWithoutConsumptionOrMutation(t *testing.T) {
	for _, spec := range []struct {
		name   string
		id     int
		char   domainchar.Character
		target int
		err    error
	}{
		{"bank full", 324, domainchar.Character{ID: 99, Level: 1, BankSlotNum: 5}, 1, pkgerrors.ErrInventoryFull},
		{"bag full", 325, domainchar.Character{ID: 99, Level: 1, BagSlotNum: 7}, 1, pkgerrors.ErrInventoryFull},
		{"pet full", 326, domainchar.Character{ID: 99, Level: 1, PetMaxNum: 180}, 1, pkgerrors.ErrInventoryFull},
		{"bag over cap", 325, domainchar.Character{ID: 99, Level: 1, BagSlotNum: 8}, 1, pkgerrors.ErrInventoryFull},
		{"level gate", 325, domainchar.Character{ID: 99, Level: 0, BagSlotNum: 1}, 1, pkgerrors.ErrInsufficientLevel},
		{"wrong owner", 325, domainchar.Character{ID: 98, Level: 1, BagSlotNum: 1}, 1, pkgerrors.ErrItemNotOwned},
		{"pet target", 326, domainchar.Character{ID: 99, Level: 1, PetMaxNum: 6}, 2, pkgerrors.ErrCannotUseItem},
	} {
		t.Run(spec.name, func(t *testing.T) {
			svc, repo := capacityTestService(t, spec.id, 2)
			char := spec.char
			_, _, err := svc.UseItem(context.Background(), &char, 88, &UseContext{TargetType: spec.target})
			if !errors.Is(err, spec.err) {
				t.Fatalf("error = %v want %v", err, spec.err)
			}
			if repo.items[88].StackCount != 2 || !reflect.DeepEqual(char, spec.char) {
				t.Fatal("rejected use changed character or stack")
			}
		})
	}
}

func TestUseMultiCapacityBooksPreflightsWholeRequest(t *testing.T) {
	for _, count := range []int{-1, 0, 2, 3, 10} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			svc, repo := capacityTestService(t, 325, 5)
			char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 5, BagSlots: 150}
			before := *char
			result, err := svc.UseMultiItem(context.Background(), char, 88, count, 1, -1)
			if count == 2 {
				if err != nil || char.BagSlotNum != 7 || char.BagSlots != 210 || repo.items[88].StackCount != 3 || result["consumed"] != 2 {
					t.Fatalf("multi use failed: char=%+v result=%v err=%v", char, result, err)
				}
				return
			}
			wantErr := pkgerrors.ErrInventoryFull
			if count <= 0 {
				wantErr = pkgerrors.ErrInvalidArgs
			} else if count > 5 {
				wantErr = pkgerrors.ErrInsufficientItems
			}
			if !errors.Is(err, wantErr) || !reflect.DeepEqual(*char, before) || repo.items[88].StackCount != 5 {
				t.Fatalf("rejected multi use changed state or wrong error: %v", err)
			}
		})
	}
}

type capacityWriteFailureRepo struct {
	domainitem.Repository
	err error
}

func (r capacityWriteFailureRepo) Delete(context.Context, int64) error           { return r.err }
func (r capacityWriteFailureRepo) UpdateStack(context.Context, int64, int) error { return r.err }

func TestUseCapacityBookWriteFailureDoesNotExpand(t *testing.T) {
	for _, stack := range []int{1, 2} {
		svc, repo := capacityTestService(t, 325, stack)
		writeErr := errors.New("write failed")
		svc.itemRepo = capacityWriteFailureRepo{Repository: repo, err: writeErr}
		char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BagSlots: 30}
		before := *char
		_, _, err := svc.UseItem(context.Background(), char, 88, nil)
		if !errors.Is(err, writeErr) || !reflect.DeepEqual(*char, before) || repo.items[88].StackCount != stack {
			t.Fatalf("write failure changed state: err=%v char=%+v", err, char)
		}
	}
}

func TestUseCapacityBookRequiresAuthoritativeTemplate(t *testing.T) {
	for _, spec := range []struct {
		name string
		tpl  *models.ItemTemplateTemplate
	}{
		{"missing", nil},
		{"wrong type", &models.ItemTemplateTemplate{ID: 325, Type: 550, UseType: 1}},
		{"not directly usable", &models.ItemTemplateTemplate{ID: 325, Type: 500, UseType: 4}},
	} {
		t.Run(spec.name, func(t *testing.T) {
			svc, repo := capacityTestService(t, 325, 2)
			manager := gamedata.NewManager(nil, zap.NewNop())
			if spec.tpl != nil {
				raw, err := json.Marshal(spec.tpl)
				if err != nil {
					t.Fatal(err)
				}
				if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
					t.Fatal(err)
				}
			}
			svc.SetGameDataManager(manager)
			char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BagSlots: 30}
			before := *char
			_, _, err := svc.UseItem(context.Background(), char, 88, nil)
			if !errors.Is(err, pkgerrors.ErrCannotUseItem) || !reflect.DeepEqual(*char, before) || repo.items[88].StackCount != 2 {
				t.Fatalf("invalid template changed state or wrong error: %v", err)
			}
		})
	}
}

func TestOtherScrollDoesNotExpandContainers(t *testing.T) {
	svc, repo := capacityTestService(t, 999999, 2)
	char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BankSlotNum: 1, PetMaxNum: 6}
	before := *char
	_, result, err := svc.UseItem(context.Background(), char, 88, nil)
	if err != nil || !reflect.DeepEqual(*char, before) || repo.items[88].StackCount != 1 || result["capacity_updates"] != nil {
		t.Fatalf("unrelated scroll behavior changed: result=%v err=%v", result, err)
	}
}
