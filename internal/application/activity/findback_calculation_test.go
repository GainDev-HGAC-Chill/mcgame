package activity

import "testing"

func TestFindbackProposedExperienceLevel80(t *testing.T) {
	for name, want := range FindbackRewardBase80 {
		if got := FindbackProposedExperience(name, 80); got != want {
			t.Fatalf("%s got %d want %d", name, got, want)
		}
	}
}
func TestFindbackMode(t *testing.T) {
	if got := FindbackAwardedExperience(434400, 1); got != 304080 {
		t.Fatalf("70%% got %d", got)
	}
	if got := FindbackAwardedExperience(434400, 2); got != 434400 {
		t.Fatalf("100%% got %d", got)
	}
	if got := FindbackAwardedExperience(434400, 3); got != 0 {
		t.Fatal("invalid mode accepted")
	}
}
func TestFindbackCost(t *testing.T) {
	if got := FindbackCost("xueyuan", 1, 2); got != 200000 {
		t.Fatalf("silver cost=%d", got)
	}
	if got := FindbackCost("xueyuan", 2, 2); got != 10 {
		t.Fatalf("gold cost=%d", got)
	}
}
