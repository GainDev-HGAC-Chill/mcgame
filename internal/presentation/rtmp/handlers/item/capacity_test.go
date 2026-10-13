package item

import (
	"encoding/json"
	"reflect"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

func TestUseExpansionBookRefreshesCapacityAndStackImmediately(t *testing.T) {
	for _, spec := range []struct {
		id    int
		field string
		want  int
	}{
		{324, "bankSlotNum", 2}, {325, "bagSlotNum", 2}, {326, "petMaxNum", 7},
	} {
		for _, stack := range []int{1, 2} {
			t.Run(spec.field+string(rune('0'+stack)), func(t *testing.T) {
				char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BagSlots: 30, BankSlotNum: 1, BankSlots: 30, PetMaxNum: 6, PetSlots: 6}
				repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: spec.id, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: stack})
				h, ctx := buildItemTestHandlerWithRepo(t, char, repo)
				loadCapacityHandlerTemplate(t, h, spec.id)
				transport := &equipCallbackTransport{}
				ctx.Connection.SetTransport(transport)
				response, err := h.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(88)})
				if err != nil || response.(map[string]interface{})["success"] != true {
					t.Fatalf("use failed: response=%v err=%v", response, err)
				}
				assertCapacityCallbacks(t, transport.calls, spec.field, spec.want, stack-1)
				persisted, err := h.charService.GetByID(ctx.Context, 99)
				if err != nil || persisted.ToDTO()[spec.field] != spec.want {
					t.Fatalf("capacity not persisted: char=%+v err=%v", persisted, err)
				}
			})
		}
	}
}

func loadCapacityHandlerTemplate(t *testing.T, h *Handler, id int) {
	t.Helper()
	raw, err := json.Marshal(models.ItemTemplateTemplate{ID: int64(id), Kind: 5, Type: 500, UseType: 1, ReqLevel: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
}

func assertCapacityCallbacks(t *testing.T, calls []callbackCall, field string, next, remaining int) {
	t.Helper()
	if len(calls) != 2 {
		t.Fatalf("unexpected callbacks: %#v", calls)
	}
	if remaining == 0 {
		if calls[0].method != "onDelCharactorSlot" || calls[0].args[0] != float64(88) {
			t.Fatalf("missing exhausted-book removal: %#v", calls[0])
		}
		switch sid := calls[0].args[1].(type) {
		case float64:
			if sid != 2104 {
				t.Fatal("wrong delete SID")
			}
		case int64:
			if sid != 2104 {
				t.Fatal("wrong delete SID")
			}
		default:
			t.Fatalf("unexpected SID type %T", sid)
		}
	} else {
		if calls[0].method != "onAddCharactorSlot" {
			t.Fatalf("missing updated stack: %#v", calls[0])
		}
		dto := calls[0].args[0].(map[string]interface{})
		if dto["id"] != int64(88) || dto["sid"] != 2104 || dto["stackNum"] != remaining {
			t.Fatalf("wrong stack update: %v", dto)
		}
	}
	if calls[1].method != "onUPP" || !reflect.DeepEqual(calls[1].args, []interface{}{map[string]interface{}{field: next}}) {
		t.Fatalf("wrong capacity refresh: %#v", calls[1])
	}
}

func TestUseExpansionBookAtCapDoesNotRefreshOrConsume(t *testing.T) {
	char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 7, BagSlots: 210}
	repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: 325, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 2})
	h, ctx := buildItemTestHandlerWithRepo(t, char, repo)
	loadCapacityHandlerTemplate(t, h, 325)
	transport := &equipCallbackTransport{}
	ctx.Connection.SetTransport(transport)
	_, err := h.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(88)})
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.calls) != 0 || repo.items[88].StackCount != 2 || char.BagSlotNum != 7 {
		t.Fatal("rejected capacity use changed state or emitted success callbacks")
	}
}

func TestUseMultiExpansionBooksRefreshesCapacityAndStack(t *testing.T) {
	for _, stack := range []int{2, 3} {
		char := &domainchar.Character{ID: 99, Level: 1, BagSlotNum: 1, BagSlots: 30}
		repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: 325, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: stack})
		h, ctx := buildItemTestHandlerWithRepo(t, char, repo)
		loadCapacityHandlerTemplate(t, h, 325)
		transport := &equipCallbackTransport{}
		ctx.Connection.SetTransport(transport)
		response, err := h.UseMultiItem(ctx, []interface{}{float64(1), float64(-1), float64(88), float64(2)})
		if err != nil || response.(map[string]interface{})["success"] != true {
			t.Fatalf("multi use failed: response=%v err=%v", response, err)
		}
		assertCapacityCallbacks(t, transport.calls, "bagSlotNum", 3, stack-2)
		if char.BagSlotNum != 3 || char.BagSlots != 90 {
			t.Fatal("multi use did not expand requested tabs")
		}
		if _, exists := response.(map[string]interface{})["result"].(map[string]interface{})["used_item"]; exists {
			t.Fatal("internal item pointer leaked into AMF response")
		}
	}
}
