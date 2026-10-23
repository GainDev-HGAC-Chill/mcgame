// Open-sourced by BaoLT

package shop

import "testing"

func TestShopColorCodeFromQuality(t *testing.T) {
	for _, test := range []struct {
		quality float64
		want    int
	}{
		{quality: 0, want: 0},
		{quality: 1, want: 1},
		{quality: 5, want: 1},
		{quality: 6, want: 2},
		{quality: 25, want: 5},
	} {
		got, err := shopColorCodeFromQuality(test.quality)
		if err != nil || got != test.want {
			t.Fatalf("quality %v = %d, %v; want %d", test.quality, got, err, test.want)
		}
	}
	for _, quality := range []float64{-1, 1.5, 26} {
		if _, err := shopColorCodeFromQuality(quality); err == nil {
			t.Fatalf("quality %v accepted", quality)
		}
	}
}
