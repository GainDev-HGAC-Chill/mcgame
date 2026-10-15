package item

import (
	"testing"

	domainitem "mcgame-server/internal/domain/item"
)

func TestDecodeTempBagClientPosition(t *testing.T) {
	tests := []struct {
		raw  int
		want int
		ok   bool
	}{
		{raw: 1, want: 1, ok: true},
		{raw: 24, want: 24, ok: true},
		{raw: 101, want: 25, ok: true},
		{raw: 124, want: 48, ok: true},
		{raw: 0, ok: false},
		{raw: 100, ok: false},
		{raw: 125, ok: false},
	}
	for _, tc := range tests {
		got, ok := decodeTempBagClientPosition(tc.raw)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("raw %d = (%d,%v), want (%d,%v)", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseAddItemFromTempArgs(t *testing.T) {
	position, slotType, slotIndex, count, ok := parseAddItemFromTempArgs([]interface{}{float64(1), float64(2106)})
	if !ok || position != 1 || slotType != domainitem.SlotTypeBag || slotIndex != 5 || count != 0 {
		t.Fatalf("parsed = (%d,%v,%d,%d,%v)", position, slotType, slotIndex, count, ok)
	}
	position, slotType, slotIndex, count, ok = parseAddItemFromTempArgs([]interface{}{float64(101), float64(2312), float64(3)})
	if !ok || position != 25 || slotType != domainitem.SlotTypeQuestBag || slotIndex != 1 || count != 3 {
		t.Fatalf("paged parsed = (%d,%v,%d,%d,%v)", position, slotType, slotIndex, count, ok)
	}
	if _, _, _, _, ok := parseAddItemFromTempArgs([]interface{}{float64(1), float64(10)}); ok {
		t.Fatal("equipment target accepted")
	}
}

func TestTempBagPayloadCapacity(t *testing.T) {
	payload := map[string]interface{}{
		"tempSlotNum": 50,
		"mx": map[string]interface{}{
			"tempSlotNum": 24,
		},
	}
	if got := tempBagPayloadCapacity(payload, false); got != 50 {
		t.Fatalf("normal capacity = %d", got)
	}
	if got := tempBagPayloadCapacity(payload, true); got != 24 {
		t.Fatalf("mx capacity = %d", got)
	}
}
