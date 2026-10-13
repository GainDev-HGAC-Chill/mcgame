package item

import (
	"context"
	"encoding/json"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

type equipCallbackTransport struct{ calls []callbackCall }

func (s *equipCallbackTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	s.calls = append(s.calls, callbackCall{method: method, args: args})
	return nil
}
func (s *equipCallbackTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return s.WriteCommand(ctx, method, args)
}
func (s *equipCallbackTransport) WriteResult(context.Context, int64, interface{}) error { return nil }
func (s *equipCallbackTransport) WriteError(context.Context, int64, error) error        { return nil }
func (s *equipCallbackTransport) Kind() string                                          { return "test" }
func (s *equipCallbackTransport) Close() error                                          { return nil }

func TestEquipOnImmediatelyRefreshesResolvedSlotAndDisplacedItem(t *testing.T) {
	for _, spec := range []struct {
		name     string
		position int
		occupied []int
		sid      int
	}{
		{"empty accessory", 1, nil, 1},
		{"replace accessory", 1, []int{0}, 1},
		{"first ring", 8, nil, 8},
		{"second ring", 8, []int{7}, 9},
		{"replace ring", 8, []int{7, 8}, 8},
		{"collection glove", 13, nil, 13},
	} {
		t.Run(spec.name, func(t *testing.T) {
			durability := 100
			incoming := &domainitem.Item{ID: 100, CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeBag, SlotIndex: 15, StackCount: 1, Durability: &durability, MaxDurability: &durability, ColorCode: 4}
			items := []*domainitem.Item{incoming}
			for _, slot := range spec.occupied {
				items = append(items, &domainitem.Item{ID: int64(200 + slot), CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped, SlotIndex: slot, StackCount: 1, Durability: &durability, MaxDurability: &durability})
			}
			repo := newSharedItemTestRepo(items...)
			h, ctx := buildItemTestHandlerWithRepo(t, &domainchar.Character{ID: 99, Level: 20, BagSlotNum: 1}, repo)
			raw, _ := json.Marshal(models.EquiptTemplateTemplate{ID: 2001, Position: float64(spec.position)})
			if err := h.gameData.GetCache().LoadTable(models.TableEquiptTemplate, []json.RawMessage{raw}); err != nil {
				t.Fatal(err)
			}
			transport := &equipCallbackTransport{}
			ctx.Connection.SetTransport(transport)
			if _, err := h.EquipOn(ctx, []interface{}{float64(incoming.ID)}); err != nil {
				t.Fatal(err)
			}
			assertEquipmentCallbacks(t, transport.calls, 100, 2116, spec.sid)
			if len(spec.occupied) > 0 && spec.sid == spec.occupied[0]+1 {
				id := int64(200 + spec.occupied[0])
				assertEquipmentCallbacks(t, transport.calls, id, spec.sid, repo.items[id].CalculateSID())
			}
		})
	}
}

func assertEquipmentCallbacks(t *testing.T, calls []callbackCall, id int64, oldSID, newSID int) {
	t.Helper()
	deleted, updated, added := false, false, false
	seenAdd := false
	seenInstanceUpdate := false
	for _, call := range calls {
		switch call.method {
		case "onDelCharactorSlot":
			if seenAdd || seenInstanceUpdate {
				t.Fatal("old-position deletion after instance update or final addition")
			}
			if call.args[0] == float64(id) {
				deleted = call.args[1] == oldSID
			}
		case "onChangeEquiptInst":
			seenInstanceUpdate = true
			dto := call.args[0].(map[string]interface{})
			if dto["id"] == id {
				updated = true
			}
		case "onAddCharactorSlot":
			seenAdd = true
			dto := call.args[0].(map[string]interface{})
			if dto["id"] == id {
				added = dto["sid"] == newSID
				if !updated {
					t.Fatal("slot added before instance update")
				}
			}
		}
	}
	if !deleted || !updated || !added {
		t.Fatalf("id%d refresh incomplete: delete=%v update=%v add=%v calls=%#v", id, deleted, updated, added, calls)
	}
}

func TestEquipOffImmediatelyRefreshesFinalBagSlot(t *testing.T) {
	for _, toSID := range []int{-1, 2127} {
		durability := 100
		it := &domainitem.Item{ID: 100, CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 0, StackCount: 1, Durability: &durability, MaxDurability: &durability}
		repo := newSharedItemTestRepo(it)
		h, ctx := buildItemTestHandlerWithRepo(t, &domainchar.Character{ID: 99, Level: 20, BagSlotNum: 1}, repo)
		transport := &equipCallbackTransport{}
		ctx.Connection.SetTransport(transport)
		if _, err := h.EquipOff(ctx, []interface{}{float64(it.ID), float64(toSID)}); err != nil {
			t.Fatal(err)
		}
		assertEquipmentCallbacks(t, transport.calls, it.ID, 1, repo.items[it.ID].CalculateSID())
	}
}

type detachedEquipTestRepo struct{ *sharedItemTestRepo }

func (r *detachedEquipTestRepo) FindByID(ctx context.Context, id int64) (*domainitem.Item, error) {
	it, err := r.sharedItemTestRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	copy := *it
	return &copy, nil
}

func TestUseItemAutoEquipRefreshesFinalSlotWithoutRelog(t *testing.T) {
	for _, occupied := range []bool{false, true} {
		durability := 100
		incoming := &domainitem.Item{ID: 100, CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeBag, SlotIndex: 15, StackCount: 1, Durability: &durability, MaxDurability: &durability}
		items := []*domainitem.Item{incoming}
		if occupied {
			items = append(items, &domainitem.Item{ID: 101, CharacterID: 99, TemplateID: 2001, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 0, StackCount: 1, Durability: &durability, MaxDurability: &durability})
		}
		repo := &detachedEquipTestRepo{newSharedItemTestRepo(items...)}
		h, ctx := buildItemTestHandlerWithRepo(t, &domainchar.Character{ID: 99, Level: 20, BagSlotNum: 1}, repo)
		transport := &equipCallbackTransport{}
		ctx.Connection.SetTransport(transport)
		if _, err := h.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(100)}); err != nil {
			t.Fatal(err)
		}
		assertEquipmentCallbacks(t, transport.calls, 100, 2116, 1)
		if occupied {
			assertEquipmentCallbacks(t, transport.calls, 101, 1, repo.items[101].CalculateSID())
		}
	}
}
