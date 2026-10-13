package item

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	appchar "mcgame-server/internal/application/character"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"

	"go.uber.org/zap"
)

type capacityCharacterWriteFailureRepo struct {
	domainchar.Repository
	persisted domainchar.Character
	writes    int
	err       error
}

func (r *capacityCharacterWriteFailureRepo) FindByID(context.Context, int64) (*domainchar.Character, error) {
	char := r.persisted
	return &char, nil
}

func (r *capacityCharacterWriteFailureRepo) Update(context.Context, *domainchar.Character) error {
	r.writes++
	return r.err
}

func TestCapacityCharacterWriteFailureReturnsErrorWithoutSuccessCallbacks(t *testing.T) {
	for _, id := range []int{324, 325, 326} {
		for _, multi := range []bool{false, true} {
			t.Run(strconv.Itoa(id)+"/multi="+strconv.FormatBool(multi), func(t *testing.T) {
				char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BagSlots: 30, BankSlotNum: 1, BankSlots: 30, PetMaxNum: 6, PetSlots: 6}
				itemRepo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: id, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 4})
				h, ctx := buildItemTestHandlerWithRepo(t, char, itemRepo)
				loadCapacityHandlerTemplate(t, h, id)
				writeErr := errors.New("character write failed")
				charRepo := &capacityCharacterWriteFailureRepo{persisted: *char, err: writeErr}
				h.charService = appchar.NewService(charRepo, zap.NewNop())
				transport := &equipCallbackTransport{}
				ctx.Connection.SetTransport(transport)
				args := []interface{}{float64(1), float64(-1), float64(88)}
				var response interface{}
				var err error
				consumed := 1
				if multi {
					consumed = 2
					response, err = h.UseMultiItem(ctx, append(args, float64(consumed)))
				} else {
					response, err = h.UseItem(ctx, args)
				}
				if response != nil || !errors.Is(err, writeErr) || !strings.Contains(err.Error(), "book preservation is not guaranteed") {
					t.Fatalf("expected explicit persistence error: response=%v err=%v", response, err)
				}
				if len(transport.calls) != 0 {
					t.Fatalf("failed persistence emitted success callbacks: %#v", transport.calls)
				}
				if charRepo.writes != 1 || itemRepo.items[88].StackCount != 4-consumed {
					t.Fatal("unexpected replay, extra consumption or rollback")
				}
				if charRepo.persisted.BagSlotNum != 1 || charRepo.persisted.BankSlotNum != 1 || charRepo.persisted.PetMaxNum != 6 {
					t.Fatal("failed write persisted capacity")
				}
			})
		}
	}
}

func TestUnrelatedItemCharacterWriteFailurePreservesExistingBehavior(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(strconv.FormatBool(multi), func(t *testing.T) {
			char := &domainchar.Character{ID: 99, Level: 1}
			repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: 1001, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 4})
			h, ctx := buildItemTestHandlerWithRepo(t, char, repo)
			charRepo := &capacityCharacterWriteFailureRepo{persisted: *char, err: errors.New("character write failed")}
			h.charService = appchar.NewService(charRepo, zap.NewNop())
			transport := &equipCallbackTransport{}
			ctx.Connection.SetTransport(transport)
			args := []interface{}{float64(1), float64(-1), float64(88)}
			var response interface{}
			var err error
			consumed := 1
			if multi {
				consumed = 2
				response, err = h.UseMultiItem(ctx, append(args, float64(consumed)))
			} else {
				response, err = h.UseItem(ctx, args)
			}
			if err != nil || response.(map[string]interface{})["success"] != true || repo.items[88].StackCount != 4-consumed || charRepo.writes != 1 {
				t.Fatalf("unrelated use behavior changed: response=%v err=%v", response, err)
			}
			for _, call := range transport.calls {
				if call.method == "onUPP" {
					t.Fatal("unrelated use emitted capacity update")
				}
			}
		})
	}
}
