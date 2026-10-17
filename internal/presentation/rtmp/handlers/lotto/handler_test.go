// Open-sourced by BaoLT

package lotto

import (
	"testing"

	domainitem "mcgame-server/internal/domain/item"
)

type lottoCallbackCall struct {
	method string
	args   []interface{}
}

type lottoCallbackRecorder struct {
	calls []lottoCallbackCall
}

func (r *lottoCallbackRecorder) SendCallback(method string, args ...interface{}) error {
	r.calls = append(r.calls, lottoCallbackCall{method: method, args: args})
	return nil
}

type lottoItemDTOBuilder struct{}

func (lottoItemDTOBuilder) BuildClientItemDTO(item *domainitem.Item) map[string]interface{} {
	return map[string]interface{}{
		"id":       item.ID,
		"sid":      item.CalculateSID(),
		"tid":      item.TemplateID,
		"stackNum": item.StackCount,
	}
}

func TestSendGrantedItemsPushesFinalInventorySlotsAndDeduplicatesStacks(t *testing.T) {
	handler := &Handler{itemDTOBuilder: lottoItemDTOBuilder{}}
	recorder := &lottoCallbackRecorder{}
	first := &domainitem.Item{ID: 10, TemplateID: 100, SlotType: domainitem.SlotTypeBag, SlotIndex: 2, StackCount: 1}
	final := &domainitem.Item{ID: 10, TemplateID: 100, SlotType: domainitem.SlotTypeBag, SlotIndex: 2, StackCount: 4}
	second := &domainitem.Item{ID: 11, TemplateID: 101, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 2}

	handler.sendGrantedItems(recorder, []*domainitem.Item{first, final, nil, second})

	if len(recorder.calls) != 2 {
		t.Fatalf("callback count = %d, want 2", len(recorder.calls))
	}
	if recorder.calls[0].method != "onAddCharactorSlot" || recorder.calls[1].method != "onAddCharactorSlot" {
		t.Fatalf("callbacks = %#v", recorder.calls)
	}
	firstDTO := recorder.calls[0].args[0].(map[string]interface{})
	if firstDTO["stackNum"] != 4 || firstDTO["sid"] != final.CalculateSID() {
		t.Fatalf("first DTO = %#v", firstDTO)
	}
	secondDTO := recorder.calls[1].args[0].(map[string]interface{})
	if secondDTO["id"] != int64(11) || secondDTO["stackNum"] != 2 {
		t.Fatalf("second DTO = %#v", secondDTO)
	}
}
