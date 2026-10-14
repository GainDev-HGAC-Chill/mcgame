package gamedata

import (
	"context"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type petConfigTestService struct {
	characterID int64
	petID       int64
	pet         *domainpet.Pet
}

func (s *petConfigTestService) GetPetDetail(_ context.Context, characterID int64, petID int64) (*domainpet.Pet, error) {
	s.characterID = characterID
	s.petID = petID
	return s.pet, nil
}

func TestGetDataConfigReturnsOwnedPetTooltipData(t *testing.T) {
	petService := &petConfigTestService{pet: &domainpet.Pet{
		ID:          123,
		CharacterID: 7,
		TemplateID:  91,
		Name:        "Pet",
		GrowRate:    1.5,
		GrowRateAdd: 0.13,
		Property: map[string]interface{}{
			"finalStrength": 21,
			"skill1":        7001,
		},
	}}
	handler := &Handler{petService: petService, logger: zap.NewNop()}
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "7"}

	result, err := handler.GetDataConfig(ctx, []interface{}{float64(39), float64(123)})
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]interface{})
	data := response["data"].(map[string]interface{})
	if petService.characterID != 7 || petService.petID != 123 {
		t.Fatalf("lookup = character %d pet %d", petService.characterID, petService.petID)
	}
	if response["type"] != 39 || response["index"] != 123 {
		t.Fatalf("response envelope = %#v", response)
	}
	if data["id"] != int64(123) || data["skill1"] != 7001 || data["attStrength"] != 21 {
		t.Fatalf("pet tooltip data = %#v", data)
	}
	if data["growRate"] != 1.5 || data["growRateAdd"] != 0.13 {
		t.Fatalf("pet growth data = %#v", data)
	}
}

func TestGetDataConfigReturnsEmptyPetDataWithoutSession(t *testing.T) {
	handler := &Handler{logger: zap.NewNop()}
	ctx := &rtmp.RPCContext{Context: context.Background()}

	result, err := handler.GetDataConfig(ctx, []interface{}{float64(39), float64(123)})
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]interface{})
	if response["data"] != nil || response["index"] != 123 {
		t.Fatalf("response = %#v", response)
	}
}
