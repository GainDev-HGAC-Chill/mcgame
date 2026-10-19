// Open-sourced by BaoLT

package shop

import (
	"strings"
	"testing"

	"mcgame-server/internal/gamedata/models"
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
