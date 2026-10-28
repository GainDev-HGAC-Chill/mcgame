package combat

import "testing"

func TestCreatureIsCatchableUsesPositiveCatalogValues(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  bool
	}{
		{name: "zero", value: 0, want: false},
		{name: "negative", value: -1, want: false},
		{name: "normal", value: 1, want: true},
		{name: "quest pet", value: 2, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := creatureIsCatchable(test.value); got != test.want {
				t.Fatalf("creatureIsCatchable(%v) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
