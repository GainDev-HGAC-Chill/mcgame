package item

import "testing"

func TestPetBagDescriptionQualityPreservesGrowthDecimals(t *testing.T) {
	for _, spec := range []struct {
		text string
		want int
	}{
		{"Chứa pet phẩm chất 1.5", 15},
		{"Chứa pet phẩm chất\n 1.5", 15},
		{"PHẨM CHẤT 1,5", 15},
		{"Pham chat 1.5", 15},
		{"Chứa 1 pet Chiến Dương 1.5 (Cấp độ mang theo 50)", 15},
		{"Chứa 1 Camy 1.5 xanh dương", 15},
		{"Chứa pet phẩm chất 1.9", 19},
		{"Chứa pet phẩm chất 2.5", 25},
		{"Chứa pet cấp 50", 0},
	} {
		if got := parseDescriptionQuality(spec.text); got != spec.want {
			t.Fatalf("%q quality=%d want%d", spec.text, got, spec.want)
		}
	}
}
