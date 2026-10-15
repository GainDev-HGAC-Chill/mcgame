package item

import (
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
)

func TestIntimacyItemEffectsMatchDescriptions(t *testing.T) {
	tests := []struct {
		id           int
		amount       int
		oppositeOnly bool
	}{
		{id: 1302, amount: 60},
		{id: 1303, amount: 60, oppositeOnly: true},
		{id: 1315, amount: 660},
		{id: 32, amount: 660, oppositeOnly: true},
	}
	for _, test := range tests {
		amount, oppositeOnly, ok := intimacyItemEffect(test.id)
		if !ok || amount != test.amount || oppositeOnly != test.oppositeOnly {
			t.Fatalf("template %d effect = %d/%v/%v", test.id, amount, oppositeOnly, ok)
		}
	}
}

func TestSummonBookPetEligibilityRequiresBoundQuality(t *testing.T) {
	pet := &domainpet.Pet{
		Property:     map[string]interface{}{"binded": 1},
		CreatureData: map[string]interface{}{"qLevel": 3},
	}
	if !summonBookPetEligible(pet, 3) {
		t.Fatal("bound orange pet should be eligible")
	}
	pet.CreatureData["qLevel"] = 2
	if summonBookPetEligible(pet, 3) || !summonBookPetEligible(pet, 2) {
		t.Fatal("purple pet quality gate mismatch")
	}
	pet.Property["binded"] = 0
	if summonBookPetEligible(pet, 2) {
		t.Fatal("unbound pet should be rejected")
	}
}

func TestSecretPassageBookRegistry(t *testing.T) {
	if !isSecretPassageBook(128) || !isSecretPassageBook(2137) || isSecretPassageBook(2514) {
		t.Fatal("secret passage book registry mismatch")
	}
}

func TestPopularityTargetItemRegistry(t *testing.T) {
	if !isPopularityTargetItem(724) || !isPopularityTargetItem(783) || isPopularityTargetItem(1303) {
		t.Fatal("popularity target item registry mismatch")
	}
}
