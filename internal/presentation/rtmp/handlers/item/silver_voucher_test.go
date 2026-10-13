package item

import (
	"encoding/json"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

func TestSilverVoucherUpdatesCorrectWalletOnSingleAndMultiUse(t *testing.T) {
	for _, multi := range []bool{false, true} {
		for _, bound := range []bool{false, true} {
			char := &domainchar.Character{ID: 99, Level: 1, Money: 7, MoneyBind: 9}
			repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: 1132, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 2, IsBound: bound})
			h, ctx := buildItemTestHandlerWithRepo(t, char, repo)
			raw, err := json.Marshal(models.ItemTemplateTemplate{ID: 1132, Type: 550, UseType: 1, ReqLevel: 1})
			if err != nil {
				t.Fatal(err)
			}
			if err := h.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
				t.Fatal(err)
			}
			transport := &equipCallbackTransport{}
			ctx.Connection.SetTransport(transport)
			delta := float64(100000)
			if multi {
				delta = 200000
				_, err = h.UseMultiItem(ctx, []interface{}{float64(1), float64(-1), float64(88), float64(2)})
			} else {
				_, err = h.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(88)})
			}
			if err != nil {
				t.Fatal(err)
			}
			key, total := "money", float64(7)+delta
			if bound {
				key, total = "moneyBind", float64(9)+delta
			}
			count := 0
			for _, call := range transport.calls {
				if call.method == "onAddMoney" {
					count++
					if len(call.args) != 4 || call.args[1] != key || call.args[2] != delta || call.args[3] != total {
						t.Fatalf("wrong currency callback: %+v", call)
					}
				}
			}
			if count != 1 {
				t.Fatalf("currency callback count=%d calls=%+v", count, transport.calls)
			}
		}
	}
}
