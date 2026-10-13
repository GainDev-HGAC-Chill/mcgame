package shop

import (
	"encoding/json"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestResolveNPCShopIDRoutesGatheringGloveButton(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, err := json.Marshal(models.NpcTemplate{ID: 275, ShopID: 32})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableNpc, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	shopID, ok := resolveNPCShopID(manager, 275)
	if !ok || shopID != 32 {
		t.Fatalf("shopID=%d ok=%v", shopID, ok)
	}
}

func TestResolveNPCShopIDRejectsUnknownNPC(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	if shopID, ok := resolveNPCShopID(manager, 275); ok || shopID != 0 {
		t.Fatalf("shopID=%d ok=%v", shopID, ok)
	}
}
