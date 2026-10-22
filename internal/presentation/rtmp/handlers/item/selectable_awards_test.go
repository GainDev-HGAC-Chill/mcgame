package item

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
)

type selectableConsumeFailRepo struct {
	*itemTestRepo
	failID int64
}

func (r *selectableConsumeFailRepo) Delete(ctx context.Context, id int64) error {
	if id == r.failID {
		return errors.New("forced selectable package consume failure")
	}
	return r.itemTestRepo.Delete(ctx, id)
}

func TestConfiguredSelectableAwardOpensPanelAndConsumesAfterChoice(t *testing.T) {
	char := &domainchar.Character{ID: 7, Name: "tester", Level: 80, BagSlots: 30, BagSlotNum: 1}
	packageItem := &domainitem.Item{ID: 90, CharacterID: char.ID, TemplateID: 3376, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
	handler, ctx := buildItemTestHandler(t, char, packageItem)
	transport := &equipCallbackTransport{}
	ctx.Connection.SetTransport(transport)

	loadItemHandlerGiftTemplate(t, handler, models.ItemTemplateTemplate{ID: 3376, Name: "Túi Quà Cánh Xanh Dương", Type: 550, UseType: 1})
	loadItemHandlerEquipmentTemplate(t, handler, models.EquiptTemplateTemplate{ID: 1676, Name: "Cánh Thiên Sứ", Kind: 13, Type: 1300, Position: 14})
	loadItemHandlerAwardTemplates(t, handler,
		configuredHandlerAward(1, 3376, 1676, 0),
		configuredHandlerAward(2, 3376, 1678, 1),
		configuredHandlerAward(3, 3376, 1677, 2),
	)

	if _, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(packageItem.ID)}); err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}
	if len(transport.calls) != 1 || transport.calls[0].method != "itemMultitoSel" {
		t.Fatalf("open callbacks = %#v", transport.calls)
	}
	openPayload, ok := transport.calls[0].args[0].(map[string]interface{})
	if !ok {
		t.Fatalf("open payload = %#v", transport.calls[0].args)
	}
	choices, ok := openPayload["packItemList"].(map[string]interface{})
	if !ok || len(choices) != 3 {
		t.Fatalf("packItemList = %#v", openPayload["packItemList"])
	}
	first := choices["0"].(map[string]interface{})
	if first["ti"] != 19 || first["ii"] != 1676 || first["n"] != 1 {
		t.Fatalf("first choice = %#v", first)
	}

	transport.calls = nil
	response, err := handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
	if err != nil {
		t.Fatalf("SelMultiItemByIdx() error = %v", err)
	}
	if response != 0 {
		t.Fatalf("response = %#v, want 0", response)
	}
	if len(transport.calls) != 2 || transport.calls[0].method != "onAddCharactorSlot" || transport.calls[1].method != "onDelCharactorSlot" {
		t.Fatalf("selection callbacks = %#v", transport.calls)
	}
	if _, err := handler.itemService.GetItemByID(ctx.Context, char.ID, packageItem.ID); err == nil {
		t.Fatal("completed package remained in inventory")
	}
}

func TestConfiguredSelectableAwardRejectsStaleAndInvalidSelections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(repo *itemTestRepo, packageItem *domainitem.Item)
		args   func(packageItem *domainitem.Item) []interface{}
	}{
		{
			name: "wrong item instance",
			args: func(packageItem *domainitem.Item) []interface{} {
				return []interface{}{float64(packageItem.ID + 1), float64(0)}
			},
		},
		{
			name: "bad index",
			args: func(packageItem *domainitem.Item) []interface{} {
				return []interface{}{float64(packageItem.ID), float64(9)}
			},
		},
		{
			name: "missing package",
			mutate: func(repo *itemTestRepo, packageItem *domainitem.Item) {
				delete(repo.items, packageItem.ID)
			},
			args: func(packageItem *domainitem.Item) []interface{} {
				return []interface{}{float64(packageItem.ID), float64(0)}
			},
		},
		{
			name: "changed package template",
			mutate: func(repo *itemTestRepo, packageItem *domainitem.Item) {
				repo.items[packageItem.ID].TemplateID = 3846
			},
			args: func(packageItem *domainitem.Item) []interface{} {
				return []interface{}{float64(packageItem.ID), float64(0)}
			},
		},
	}

	for _, spec := range tests {
		t.Run(spec.name, func(t *testing.T) {
			handler, ctx, repo, packageItem, transport := openConfiguredSelectableAward(t, 30)
			transport.calls = nil
			if spec.mutate != nil {
				spec.mutate(repo, packageItem)
			}
			response, err := handler.SelMultiItemByIdx(ctx, spec.args(packageItem))
			if err != nil {
				t.Fatalf("SelMultiItemByIdx() error = %v", err)
			}
			if response != -1 {
				t.Fatalf("response = %#v, want -1", response)
			}
			if len(transport.calls) != 0 {
				t.Fatalf("rejected selection callbacks = %#v", transport.calls)
			}
		})
	}
}

func TestConfiguredSelectableAwardCapacityFailureDoesNotGrantOrConsume(t *testing.T) {
	handler, ctx, repo, packageItem, transport := openConfiguredSelectableAward(t, 30)
	for index := 1; index < 30; index++ {
		id := int64(100 + index)
		repo.items[id] = &domainitem.Item{
			ID: id, CharacterID: 7, TemplateID: 5000 + index,
			ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag,
			SlotIndex: index, StackCount: 1,
		}
	}
	transport.calls = nil

	response, err := handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
	if err != nil {
		t.Fatalf("SelMultiItemByIdx() error = %v", err)
	}
	if response != -1 {
		t.Fatalf("response = %#v, want -1", response)
	}
	if len(transport.calls) != 0 {
		t.Fatalf("capacity failure callbacks = %#v", transport.calls)
	}
	if repo.items[packageItem.ID] == nil || repo.items[packageItem.ID].StackCount != 1 {
		t.Fatal("capacity failure consumed the package")
	}
	for _, item := range repo.items {
		if item.TemplateID == 1676 {
			t.Fatal("capacity failure granted the selected equipment")
		}
	}
}

func TestConfiguredSelectableAwardConsumeFailureEmitsNoSuccessAndCannotReplay(t *testing.T) {
	char := &domainchar.Character{ID: 7, Name: "tester", Level: 80, BagSlots: 30, BagSlotNum: 1}
	packageItem := &domainitem.Item{ID: 90, CharacterID: char.ID, TemplateID: 3376, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
	baseRepo := newItemTestRepo(packageItem)
	repo := &selectableConsumeFailRepo{itemTestRepo: baseRepo, failID: packageItem.ID}
	handler, ctx := buildItemTestHandlerWithRepo(t, char, repo)
	transport := &equipCallbackTransport{}
	ctx.Connection.SetTransport(transport)
	loadConfiguredSelectableTemplates(t, handler)

	if _, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(packageItem.ID)}); err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}
	transport.calls = nil
	response, err := handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
	if err != nil {
		t.Fatalf("SelMultiItemByIdx() error = %v", err)
	}
	if response != -1 || len(transport.calls) != 0 {
		t.Fatalf("consume failure response=%#v callbacks=%#v", response, transport.calls)
	}
	if baseRepo.items[packageItem.ID] == nil {
		t.Fatal("forced consume failure removed the package")
	}
	granted := 0
	for _, item := range baseRepo.items {
		if item.TemplateID == 1676 {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("granted equipment count = %d, want 1", granted)
	}

	response, err = handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
	if err != nil || response != -1 || len(transport.calls) != 0 {
		t.Fatalf("replay response=%#v error=%v callbacks=%#v", response, err, transport.calls)
	}
	granted = 0
	for _, item := range baseRepo.items {
		if item.TemplateID == 1676 {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("replay granted equipment count = %d, want 1", granted)
	}
}

func TestConfiguredSelectableAwardRejectsDuplicateSelection(t *testing.T) {
	handler, ctx, _, packageItem, transport := openConfiguredSelectableAward(t, 30)
	transport.calls = nil
	response, err := handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
	if err != nil || response != 0 {
		t.Fatalf("first selection response=%#v error=%v", response, err)
	}
	transport.calls = nil
	response, err = handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
	if err != nil || response != -1 || len(transport.calls) != 0 {
		t.Fatalf("duplicate response=%#v error=%v callbacks=%#v", response, err, transport.calls)
	}
}

func TestConfiguredSelectableAwardSerializesConcurrentSelectionPerCharacter(t *testing.T) {
	handler, ctx, repo, packageItem, transport := openConfiguredSelectableAward(t, 30)
	transport.calls = nil
	responses := make(chan interface{}, 2)
	errors := make(chan error, 2)
	for range 2 {
		go func() {
			response, err := handler.SelMultiItemByIdx(ctx, []interface{}{float64(packageItem.ID), float64(0)})
			responses <- response
			errors <- err
		}()
	}
	seen := map[interface{}]int{}
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatalf("SelMultiItemByIdx() error = %v", err)
		}
		seen[<-responses]++
	}
	if seen[0] != 1 || seen[-1] != 1 {
		t.Fatalf("responses = %#v, want one success and one rejection", seen)
	}
	granted := 0
	for _, item := range repo.items {
		if item.TemplateID == 1676 {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("concurrent selection granted %d equipment, want 1", granted)
	}
}

func openConfiguredSelectableAward(t *testing.T, bagSlots int) (*Handler, *rtmp.RPCContext, *itemTestRepo, *domainitem.Item, *equipCallbackTransport) {
	t.Helper()
	char := &domainchar.Character{ID: 7, Name: "tester", Level: 80, BagSlots: bagSlots, BagSlotNum: 1}
	packageItem := &domainitem.Item{ID: 90, CharacterID: char.ID, TemplateID: 3376, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
	repo := newItemTestRepo(packageItem)
	handler, ctx := buildItemTestHandlerWithRepo(t, char, repo)
	transport := &equipCallbackTransport{}
	ctx.Connection.SetTransport(transport)
	loadConfiguredSelectableTemplates(t, handler)
	if _, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(packageItem.ID)}); err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}
	return handler, ctx, repo, packageItem, transport
}

func loadConfiguredSelectableTemplates(t *testing.T, handler *Handler) {
	t.Helper()
	loadItemHandlerGiftTemplate(t, handler, models.ItemTemplateTemplate{ID: 3376, Name: "Túi Quà Cánh Xanh Dương", Type: 550, UseType: 1})
	loadItemHandlerEquipmentTemplate(t, handler, models.EquiptTemplateTemplate{ID: 1676, Name: "Cánh Thiên Sứ", Kind: 13, Type: 1300, Position: 14})
	loadItemHandlerAwardTemplates(t, handler,
		configuredHandlerAward(1, 3376, 1676, 0),
		configuredHandlerAward(2, 3376, 1678, 1),
		configuredHandlerAward(3, 3376, 1677, 2),
	)
}

func configuredHandlerAward(id int64, itemID int, awardID int, order int) models.ItemAwardTemplate {
	return models.ItemAwardTemplate{
		ID:          id,
		ItemID:      float64(itemID),
		AwardID:     float64(awardID),
		Type:        19,
		Count:       1,
		Quality:     8,
		PreNameType: 3,
		Payload: map[string]interface{}{
			models.AwardPayloadSelectionKey: map[string]interface{}{"order": float64(order), "getNum": float64(1)},
		},
	}
}

func loadItemHandlerAwardTemplates(t *testing.T, handler *Handler, templates ...models.ItemAwardTemplate) {
	t.Helper()
	records := make([]json.RawMessage, 0, len(templates))
	for _, template := range templates {
		raw, err := json.Marshal(template)
		if err != nil {
			t.Fatalf("marshal item award: %v", err)
		}
		records = append(records, raw)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableItemAward, records); err != nil {
		t.Fatalf("load item awards: %v", err)
	}
}

func loadItemHandlerEquipmentTemplate(t *testing.T, handler *Handler, template models.EquiptTemplateTemplate) {
	t.Helper()
	raw, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("marshal equipment template: %v", err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableEquiptTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load equipment template: %v", err)
	}
}
