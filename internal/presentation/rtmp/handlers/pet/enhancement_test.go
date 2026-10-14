package pet

import (
	"math"
	"testing"

	apppet "mcgame-server/internal/application/pet"
	domainpet "mcgame-server/internal/domain/pet"
)

func TestPetPropRefreshPayloadUsesEffectiveEnhancementComponents(t *testing.T) {
	p := &domainpet.Pet{
		ID:          101,
		GrowRate:    1.56,
		GrowRateAdd: 0.40,
		Property:    map[string]interface{}{"growRateAdd": 0},
	}

	payload := petPropRefreshPayload(p)
	if payload == nil || payload["id"] != int64(101) {
		t.Fatalf("payload = %#v", payload)
	}
	prop, ok := payload["s"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload property type = %T", payload["s"])
	}
	base, baseOK := prop["growRate"].(float64)
	add, addOK := prop["growRateAdd"].(float64)
	if !baseOK || !addOK || math.Abs(base-1.56) > 1e-9 || math.Abs(add-0.40) > 1e-9 {
		t.Fatalf("growth components = %#v/%#v", prop["growRate"], prop["growRateAdd"])
	}
	if math.Abs(base+add-1.96) > 1e-9 {
		t.Fatalf("effective growth = %.2f, want 1.96", base+add)
	}
}

func TestPetPropRefreshPayloadRejectsNilPet(t *testing.T) {
	if payload := petPropRefreshPayload(nil); payload != nil {
		t.Fatalf("payload = %#v, want nil", payload)
	}
}

func TestPetStarResponseReportsFailedAttemptAndProtectionStack(t *testing.T) {
	response := petStarResponse(&apppet.PetItemUseResult{
		Pet:            &domainpet.Pet{ID: 101, UpgradeNum: 8, GrowRateAdd: 0.40},
		ItemID:         10,
		RemainingStack: 3,
		Success:        false,
		SafeItemID:     11,
		SafeRemaining:  2,
	})
	if response["flag"] != false || response["starNum"] != 8 || response["starSafeNum"] != 2 {
		t.Fatalf("response = %#v", response)
	}
}
