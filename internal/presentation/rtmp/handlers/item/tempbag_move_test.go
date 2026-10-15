package item

import "testing"

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
