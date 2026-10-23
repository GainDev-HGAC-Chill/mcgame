// Open-sourced by BaoLT

package item

import (
	"context"
	"encoding/json"
	"testing"

	appitem "mcgame-server/internal/application/item"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

type handlerPPVERewardGranter struct{}

func (handlerPPVERewardGranter) GrantKPReward(ctx context.Context, charID int64, amount int) (map[string]interface{}, map[string]interface{}, error) {
	return map[string]interface{}{"kp": amount}, map[string]interface{}{"mlv": 0}, nil
}

func TestPPVEKPBoxRewardSendsPanelCallbackNotMoney(t *testing.T) {
	char := &domainchar.Character{ID: 99, Level: 1}
	repo := newItemTestRepo(&domainitem.Item{ID: 88, CharacterID: 99, TemplateID: 7003, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 1})
	handler, ctx := buildItemTestHandlerWithRepo(t, char, repo)
	handler.itemService.SetPPVERewardGranter(handlerPPVERewardGranter{})
	itemRaw, err := json.Marshal(models.ItemTemplateTemplate{ID: 7003, Type: 999, UseType: 1})
	if err != nil {
		t.Fatal(err)
	}
	awardRaw, err := json.Marshal(models.ItemAwardTemplate{ID: 7003, ItemID: 7003, AwardID: appitem.AwardGameCurrencyPPVEKP, Type: 35, Count: 25, Rate: 100})
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{itemRaw}); err != nil {
		t.Fatal(err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableItemAward, []json.RawMessage{awardRaw}); err != nil {
		t.Fatal(err)
	}
	transport := &equipCallbackTransport{}
	ctx.Connection.SetTransport(transport)
	if _, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(88)}); err != nil {
		t.Fatal(err)
	}
	panelCallbacks := 0
	for _, call := range transport.calls {
		if call.method == "onAddMoney" {
			t.Fatalf("unexpected money callback: %+v", call)
		}
		if call.method != "onUpdatePPVEPanel" {
			continue
		}
		panelCallbacks++
		if len(call.args) != 4 || call.args[0].(map[string]interface{})["kp"] != 25 || call.args[2] != nil || call.args[3] != false {
			t.Fatalf("panel callback=%+v", call)
		}
	}
	if panelCallbacks != 1 {
		t.Fatalf("callbacks=%+v", transport.calls)
	}
}
