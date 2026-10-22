// Open-sourced by BaoLT

package shop

import (
	"encoding/json"
	"strings"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestBuildShopConfigRowsUsesFlashFieldNames(t *testing.T) {
	rows := buildShopConfigRows([]*models.ShopSlotTemplate{{
		ID: 7, Amount: -1, Gold: 2, Gt: 1, ItemID: 99, Money: 40,
		PNum1: 3, PNum2: 4, PType1: 20, PType2: 21, Point: 5,
		Position: 6, PriceAll: 1, Quality: 15, R: 8, Sale: 9,
		Sid: 10, St: 0, Type: 29,
	}})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	for _, want := range []string{
		"var obj={};", "obj.id='7';", "obj.itemId='99';", "obj.priceAll='1';",
		"obj.pType1='20';", "obj.pNum1='3';", "obj.sid='10';", "obj.type='29';",
	} {
		if !strings.Contains(rows[0], want) {
			t.Fatalf("row %q missing %q", rows[0], want)
		}
	}
	if strings.Contains(rows[0], "item_id") || strings.Contains(rows[0], "p_type1") {
		t.Fatalf("row uses database field names: %q", rows[0])
	}
}

func TestBuildShopConfigRowsSkipsNil(t *testing.T) {
	rows := buildShopConfigRows([]*models.ShopSlotTemplate{nil, {ID: 1}})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
}

func TestShopSlotHasTemplateRejectsMissingCatalogReference(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, err := json.Marshal(models.ItemTemplateTemplate{ID: 99})
	if err != nil {
		t.Fatalf("marshal item template: %v", err)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load item template: %v", err)
	}

	if !shopSlotHasTemplate(manager, &models.ShopSlotTemplate{ID: 1, Type: 29, ItemID: 99}) {
		t.Fatal("expected existing item template to be renderable")
	}
	if shopSlotHasTemplate(manager, &models.ShopSlotTemplate{ID: 2, Type: 29, ItemID: 100}) {
		t.Fatal("expected missing item template to be rejected")
	}
	if shopSlotHasTemplate(manager, &models.ShopSlotTemplate{ID: 3, Type: 999, ItemID: 99}) {
		t.Fatal("expected unsupported table type to be rejected")
	}
}

func TestMissingTemplateShopSlotSerializesQuarantineOverride(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	slot := &models.ShopSlotTemplate{ID: 2, Type: 29, ItemID: 100, Sid: 66, St: 0}
	quarantined, valid := shopSlotForClient(manager, slot)
	if valid || quarantined == nil {
		t.Fatalf("shopSlotForClient = %#v, %v; want quarantine", quarantined, valid)
	}
	rows := buildShopConfigRows([]*models.ShopSlotTemplate{quarantined})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	for _, want := range []string{"obj.id='2';", "obj.sid='0';", "obj.st='5';", "obj.itemId='100';"} {
		if !strings.Contains(rows[0], want) {
			t.Fatalf("quarantine row %q missing %q", rows[0], want)
		}
	}
	if slot.Sid != 66 || slot.St != 0 {
		t.Fatalf("source slot mutated: %#v", slot)
	}
}
