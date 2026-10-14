package item

import (
	"encoding/json"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

func TestGiftCurrencyVoucherCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name       string
		templateID int
		level      int
		clientKey  string
		amount     float64
	}{
		{"silver gift", 654, 1, "moneyBind", 10000},
		{"gold gift", 1064, 50, "goldBind", 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			char := &domainchar.Character{ID: 99, Level: tc.level, MoneyBind: 7, GoldBind: 11}
			repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: tc.templateID, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 1})
			h, ctx := buildItemTestHandlerWithRepo(t, char, repo)
			raw, err := json.Marshal(models.ItemTemplateTemplate{ID: int64(tc.templateID), Type: 550, UseType: 1, ReqLevel: float64(tc.level)})
			if err != nil {
				t.Fatal(err)
			}
			if err := h.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
				t.Fatal(err)
			}
			transport := &equipCallbackTransport{}
			ctx.Connection.SetTransport(transport)
			if _, err := h.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(88)}); err != nil {
				t.Fatal(err)
			}
			wantTotal := tc.amount
			if tc.clientKey == "moneyBind" {
				wantTotal += 7
			} else {
				wantTotal += 11
			}
			currencyCallbacks := 0
			for _, call := range transport.calls {
				if call.method != "onAddMoney" {
					continue
				}
				currencyCallbacks++
				if len(call.args) != 4 || call.args[1] != tc.clientKey || call.args[2] != tc.amount || call.args[3] != wantTotal {
					t.Fatalf("wrong currency callback: %+v", call)
				}
			}
			if currencyCallbacks != 1 {
				t.Fatalf("currency callback count=%d calls=%+v", currencyCallbacks, transport.calls)
			}
		})
	}
}
