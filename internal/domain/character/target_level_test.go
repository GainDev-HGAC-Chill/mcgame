package character

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

func TestSetTargetLevelInvalidAndSameAreUnchanged(t *testing.T) {
	for _, target := range []int{-1, 0, MaxLevel + 1, 1} {
		t.Run(strconv.Itoa(target), func(t *testing.T) {
			char := NewCharacter(uuid.New(), "target", 1, 0)
			char.Experience = 42
			before := *char
			changed, err := char.SetTargetLevel(target)
			if changed || (target != 1 && err == nil) || !reflect.DeepEqual(before, *char) {
				t.Fatalf("target=%d changed=%v err=%v mutated=%v", target, changed, err, !reflect.DeepEqual(before, *char))
			}
		})
	}
}

func TestSetTargetLevelRaiseUsesCanonicalGrowth(t *testing.T) {
	char := NewCharacter(uuid.New(), "target", 1, 0)
	char.Experience = 42
	want := *char
	want.RaiseLevelTo(MaxLevel)
	changed, err := char.SetTargetLevel(MaxLevel)
	if err != nil || !changed || !reflect.DeepEqual(want, *char) {
		t.Fatalf("canonical raise mismatch: changed=%v err=%v", changed, err)
	}
}

func TestSetTargetLevelDowngradeRefundAndPreserveAssets(t *testing.T) {
	char := NewCharacter(uuid.New(), "target", 1, 0)
	char.SetClassAptitudes(1000, 2000, 3000, 4000, 5000)
	char.AptStrengthEvolution = 17
	char.RaiseLevelTo(90)
	char.AllocateAttribute("strength")
	char.AllocateAttribute("agility")
	char.Experience = 42
	char.Gold = 999999
	char.Money = 800
	char.RebirthExp = 500
	char.GuideLog[100] = true
	want := *char
	want.Level = 5
	want.Experience = 0
	want.Strength -= 85
	want.Agility -= 85
	want.Stamina -= 85
	want.Intelligence -= 85
	want.Spirit -= 85
	want.AptStrength = 0
	want.AptAgility = 0
	want.AttrPoints = 20
	want.MaxAttrPoints = 20
	want.DistributedAttrPoints = 0
	want.LastPoint = "0"
	want.RefreshRuntimeStats()
	changed, err := char.SetTargetLevel(5)
	if err != nil || !changed || !reflect.DeepEqual(want, *char) {
		t.Fatalf("downgrade did not preserve non-progression fields: changed=%v err=%v", changed, err)
	}
	char.SetTargetLevel(1)
	if char.AttrPoints != 0 || char.MaxAttrPoints != 0 || char.Strength != 10 {
		t.Fatalf("level-one state inconsistent: %+v", char)
	}
}

func TestSetTargetLevelPreservesUnallocatedAptitude(t *testing.T) {
	char := NewCharacter(uuid.New(), "target", 1, 0)
	char.RaiseLevelTo(10)
	char.AptStrength = 1234
	if _, err := char.SetTargetLevel(5); err != nil || char.AptStrength != 1234 {
		t.Fatalf("aptitude lost: %d err=%v", char.AptStrength, err)
	}
}

func TestSetTargetLevelRejectsAmbiguousAllocation(t *testing.T) {
	char := NewCharacter(uuid.New(), "target", 1, 0)
	char.RaiseLevelTo(10)
	char.AllocateAttribute("strength")
	char.AptStrength += 1000
	before := *char
	if changed, err := char.SetTargetLevel(5); changed || err == nil || !reflect.DeepEqual(before, *char) {
		t.Fatalf("ambiguous aptitude must not be destroyed: changed=%v err=%v", changed, err)
	}
}

func TestSetTargetLevelPreservesBonusPoints(t *testing.T) {
	for _, allocated := range []bool{false, true} {
		for _, target := range []int{5, 1} {
			char := NewCharacter(uuid.New(), "target", 1, 0)
			char.RaiseLevelTo(10)
			char.MaxAttrPoints += 17
			char.AttrPoints += 17
			if allocated {
				for i := 0; i < 50; i++ {
					char.AllocateAttribute("strength")
				}
			}
			changed, err := char.SetTargetLevel(target)
			want := (target-1)*AttributePointsPerLevel + 17
			if err != nil || !changed || char.MaxAttrPoints != want || char.AttrPoints != want || char.DistributedAttrPoints != 0 {
				t.Fatalf("allocated=%v target=%d points=%d max=%d err=%v", allocated, target, char.AttrPoints, char.MaxAttrPoints, err)
			}
		}
	}
}

func TestSetTargetLevelRejectsInvalidAccountingBeforeMutation(t *testing.T) {
	for _, corrupt := range []func(*Character){
		func(c *Character) { c.MaxAttrPoints-- },
		func(c *Character) { c.AttrPoints-- },
		func(c *Character) { c.DistributedAttrPoints = -1 },
		func(c *Character) { c.DistributedAttrPoints = c.MaxAttrPoints + 1 },
	} {
		for _, target := range []int{1, 20} {
			char := NewCharacter(uuid.New(), "target", 1, 0)
			char.RaiseLevelTo(10)
			corrupt(char)
			before := *char
			if changed, err := char.SetTargetLevel(target); changed || err == nil || !reflect.DeepEqual(before, *char) {
				t.Fatalf("invalid accounting mutated: changed=%v err=%v", changed, err)
			}
		}
	}
}
