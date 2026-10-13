package item

import (
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

func TestSortedBagRefreshDeletesBeforeAddingFinalRecords(t *testing.T) {
	h, _ := buildItemTestHandler(t, &domainchar.Character{ID: 99, Level: 3, BagSlotNum: 1})
	old := map[int64]int{11: 2116, 12: 2101, 13: 2117}
	final := []*domainitem.Item{
		{ID: 11, CharacterID: 99, TemplateID: 1001, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 25},
		{ID: 12, CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeBag, SlotIndex: 1, StackCount: 1},
	}
	r := &handlerCallbackRecorder{}
	if err := h.refreshSortedBag(r, old, final); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 6 {
		t.Fatalf("calls=%d", len(r.calls))
	}
	deleted := map[int64]int{}
	for _, call := range r.calls[:3] {
		if call.method != "onDelCharactorSlot" || len(call.args) != 2 {
			t.Fatalf("expected all deletes first: %#v", call)
		}
		deleted[int64(call.args[0].(float64))] = call.args[1].(int)
	}
	for id, sid := range old {
		if deleted[id] != sid {
			t.Fatalf("wrong original SID for%d", id)
		}
	}
	for index, call := range r.calls[3:5] {
		if call.method != "onAddCharactorSlot" {
			t.Fatalf("missing final record: %#v", call)
		}
		dto := call.args[0].(map[string]interface{})
		if dto["sid"] != final[index].CalculateSID() {
			t.Fatalf("wrong final SID: %#v", dto)
		}
		if dto["id"] == dto["sid"] {
			t.Fatal("record ID replaced by SID")
		}
	}
	last := r.calls[5]
	if last.method != "onItemSort" || len(last.args) != 1 || len(last.args[0].([]interface{})) != 0 {
		t.Fatalf("expected source-free full repaint: %#v", last)
	}
}
