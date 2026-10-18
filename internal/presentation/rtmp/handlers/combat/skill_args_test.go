package combat

import "testing"

func TestParseSkillLearnIDAcceptsAMFNumericString(t *testing.T) {
	for _, test := range []struct {
		input interface{}
		want  int
	}{
		{input: "2216", want: 2216},
		{input: float64(2226), want: 2226},
		{input: 2218, want: 2218},
	} {
		got, ok := parseSkillLearnID(test.input)
		if !ok || got != test.want {
			t.Fatalf("parseSkillLearnID(%#v) = (%d, %v), want (%d, true)", test.input, got, ok, test.want)
		}
	}

	if _, ok := parseSkillLearnID("not-a-skill"); ok {
		t.Fatal("parseSkillLearnID(non-numeric string) accepted invalid input")
	}
}
